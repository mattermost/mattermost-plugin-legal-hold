package jobs

import (
	"context"
	"fmt"
	"strings"

	"sync"
	"time"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-plugin-api/cluster"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/wiggin77/merror"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/legalhold"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/kvstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
)

type LegalHoldJob struct {
	mux      sync.Mutex
	settings *LegalHoldJobSettings
	job      *cluster.Job
	runner   *runInstance

	id          string
	papi        plugin.API
	client      *pluginapi.Client
	sqlstore    *sqlstore.SQLStore
	kvstore     kvstore.KVStore
	filebackend filestore.FileBackend
}

func NewLegalHoldJob(id string, api plugin.API, client *pluginapi.Client, sqlstore *sqlstore.SQLStore, kvstore kvstore.KVStore, filebackend filestore.FileBackend) (*LegalHoldJob, error) {
	return &LegalHoldJob{
		settings:    &LegalHoldJobSettings{},
		id:          id,
		papi:        api,
		client:      client,
		sqlstore:    sqlstore,
		kvstore:     kvstore,
		filebackend: filebackend,
	}, nil
}

func (j *LegalHoldJob) GetID() string {
	return j.id
}

// OnConfigurationChange is called by the job manager whenenver the plugin settings have changed.
// Stop current job (if any) and start a new job (if enabled) with new settings.
func (j *LegalHoldJob) OnConfigurationChange(cfg *config.Configuration) error {
	j.client.Log.Debug("LegalHoldJob: Configuration Changed")
	settings, err := parseLegaHoldJobSettings(cfg)
	if err != nil {
		j.client.Log.Error(fmt.Sprintf("LegalHoldJob: Error parsing job settings: %v", err.Error()))
		return err
	}

	// stop existing job (if any)
	if err := j.Stop(time.Second * 10); err != nil {
		j.client.Log.Error("Error stopping Legal Hold job for config change", "err", err)
	}

	if settings.EnableLegalHoldJobs {
		j.client.Log.Debug("Preparing to start legal hold job.")
		return j.start(settings)
	}

	j.client.Log.Debug("Not starting Legal Hold Job as it is disabled in the config.")
	return nil
}

// start schedules a new job with specified settings.
func (j *LegalHoldJob) start(settings *LegalHoldJobSettings) error {
	j.mux.Lock()
	defer j.mux.Unlock()

	j.settings = settings

	job, err := cluster.Schedule(j.papi, j.id, j.nextWaitInterval, j.run)
	if err != nil {
		return fmt.Errorf("cannot start Legal Hold job: %w", err)
	}
	j.job = job

	j.client.Log.Debug("Legal Hold job started")

	return nil
}

// Stop stops the current job (if any). If the timeout is exceeded an error
// is returned.
func (j *LegalHoldJob) Stop(timeout time.Duration) error {
	var job *cluster.Job
	var runner *runInstance

	j.mux.Lock()
	job = j.job
	runner = j.runner
	j.job = nil
	j.runner = nil
	j.mux.Unlock()

	merr := merror.New()

	if job != nil {
		if err := job.Close(); err != nil {
			merr.Append(fmt.Errorf("error closing job: %w", err))
		}
	}

	if runner != nil {
		if err := runner.stop(timeout); err != nil {
			merr.Append(fmt.Errorf("error stopping job runner: %w", err))
		}
	}

	j.client.Log.Debug("Legal Hold Job stopped", "err", merr.ErrorOrNil())

	return merr.ErrorOrNil()
}

func (j *LegalHoldJob) getSettings() *LegalHoldJobSettings {
	j.mux.Lock()
	defer j.mux.Unlock()
	return j.settings.Clone()
}

// nextWaitInterval is called by the cluster job scheduler to determine how long to wait until the
// next job run.
func (j *LegalHoldJob) nextWaitInterval(now time.Time, metaData cluster.JobMetadata) time.Duration {
	settings := j.getSettings()

	lastFinished := metaData.LastFinished
	if lastFinished.IsZero() {
		lastFinished = now.AddDate(-1, 0, 0)
	}

	next := settings.CalcNext(lastFinished, settings.TimeOfDay)
	delta := next.Sub(now)

	j.client.Log.Debug("Legal Hold Job next run scheduled", "last", lastFinished.Format(FullLayout), "next", next.Format(FullLayout), "wait", delta.String())

	return delta
}

func (j *LegalHoldJob) RunFromAPI() {
	j.run()
}

func (j *LegalHoldJob) RunSingleLegalHold(legalHoldID string) error {
	// Retrieve the single legal hold from the store
	legalHold, err := j.kvstore.GetLegalHoldByID(legalHoldID)
	if err != nil {
		return fmt.Errorf("failed to fetch legal hold: %w", err)
	}
	if legalHold == nil {
		return fmt.Errorf("legal hold not found: %s", legalHoldID)
	}

	j.runWith([]model.LegalHold{*legalHold}, true)
	return nil
}

func (j *LegalHoldJob) run() {
	j.mux.Lock()
	oldRunner := j.runner
	j.mux.Unlock()

	if oldRunner != nil {
		j.client.Log.Error("Multiple Legal Hold jobs scheduled concurrently; there can be only one")
		return
	}

	j.client.Log.Info("Processing all Legal Holds")

	// Retrieve the legal holds from the store.
	legalHolds, err := j.kvstore.GetAllLegalHolds()
	if err != nil {
		j.client.Log.Error("Failed to fetch legal holds from store", err)
		return
	}

	j.runWith(legalHolds, false)
}

func (j *LegalHoldJob) runWith(legalHolds []model.LegalHold, forceRun bool) {
	j.client.Log.Info("Running Legal Hold Job")
	exitSignal := make(chan struct{})
	ctx, canceller := context.WithCancel(context.Background())

	runner := &runInstance{
		canceller:  canceller,
		exitSignal: exitSignal,
	}

	defer func() {
		canceller()
		close(exitSignal)

		j.mux.Lock()
		j.runner = nil
		j.mux.Unlock()
	}()

	var settings *LegalHoldJobSettings
	j.mux.Lock()
	j.runner = runner
	settings = j.settings.Clone()
	j.mux.Unlock()

	for _, lh := range legalHolds {
		now := mattermostModel.GetMillis()

		legalHold := lh.DeepCopy()

		for {
			if legalHold.IsFinished() {
				j.client.Log.Debug(fmt.Sprintf("Legal Hold %s has ended and therefore does not need another execution.", legalHold.ID))
				break
			}

			if !forceRun && !legalHold.NeedsExecuting(now) {
				j.client.Log.Debug(fmt.Sprintf("Legal Hold %s is not yet ready to be executed again.", legalHold.ID))
				break
			}
			if legalHold.LastExecutionEndedAt >= now {
				j.client.Log.Debug(fmt.Sprintf("Legal Hold %s was already executed after the current time.", legalHold.ID))
				break
			}

			j.client.Log.Debug(fmt.Sprintf("Creating Legal Hold Execution for legal hold: %s", legalHold.ID))
			lhe := legalhold.NewExecution(legalHold, j.papi, j.sqlstore, j.kvstore, j.filebackend)

			if updatedLH, err := lhe.Execute(now); err != nil {
				if strings.Contains(err.Error(), "another execution is already running") {
					j.client.Log.Debug("Another execution is already running for this legal hold", "legal_hold_id", legalHold.ID)
					break
				}
				j.client.Log.Error("An error occurred executing the legal hold.", err)
			} else {
				// Update legal hold with the new execution details (last execution time and last message)
				// Also set it to IDLE again since the execution has ended.
				stored, err := j.kvstore.GetLegalHoldByID(legalHold.ID)
				if err != nil {
					j.client.Log.Error("Failed to fetch the LegalHold prior to updating", err)
					break
				}
				legalHold = stored.DeepCopy()
				legalHold.LastExecutionEndedAt = updatedLH.LastExecutionEndedAt
				legalHold.HasMessages = updatedLH.HasMessages

				newLH, err := j.kvstore.UpdateLegalHold(legalHold, *stored)
				if err != nil {
					j.client.Log.Error("Failed to update legal hold", err)
					break
				}
				j.client.Log.Info("legal hold executed", "legal_hold_id", newLH.ID, "legal_hold_name", newLH.Name)
			}

			time.Sleep(time.Millisecond * 250)
		}

		// Set legal hold as IDLE since we are finished with it
		if err := j.kvstore.UpdateLegalHoldStatus(legalHold.ID, model.LegalHoldStatusIdle); err != nil {
			j.client.Log.Error("Failed to update legal hold status", "err", err, "legal_hold_id", legalHold.ID)
		}
	}

	_ = ctx
	_ = settings
}

type runInstance struct {
	canceller  func()        // called to stop a currently executing run
	exitSignal chan struct{} // closed when the currently executing run has exited
}

func (r *runInstance) stop(timeout time.Duration) error {
	// cancel the run
	r.canceller()

	// wait for it to exit
	select {
	case <-r.exitSignal:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("waiting on job to stop timed out after %s", timeout.String())
	}
}
