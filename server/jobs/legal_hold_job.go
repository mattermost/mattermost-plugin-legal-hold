package jobs

import (
	"context"
	"fmt"

	"strings"
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

type LegalHoldRunOnceProps struct {
	LegalHold model.LegalHold
	ForceRun  bool
}

type LegalHoldJob struct {
	mux      *cluster.Mutex
	job      *cluster.Job
	runner   *runInstance
	settings *LegalHoldJobSettings

	id          string
	papi        plugin.API
	client      *pluginapi.Client
	sqlstore    *sqlstore.SQLStore
	kvstore     kvstore.KVStore
	filebackend filestore.FileBackend

	onceScheduler *cluster.JobOnceScheduler
}

func NewLegalHoldJob(id string, api plugin.API, client *pluginapi.Client, sqlstore *sqlstore.SQLStore, kvstore kvstore.KVStore, filebackend filestore.FileBackend) (*LegalHoldJob, error) {
	scheduledJobMutex, err := cluster.NewMutex(api, "legal_hold_scheduled_job")
	if err != nil {
		return nil, fmt.Errorf("could not create mutex for Legal Hold job: %w", err)
	}

	return &LegalHoldJob{
		settings:      &LegalHoldJobSettings{},
		id:            id,
		papi:          api,
		client:        client,
		sqlstore:      sqlstore,
		kvstore:       kvstore,
		filebackend:   filebackend,
		mux:           scheduledJobMutex,
		onceScheduler: cluster.GetJobOnceScheduler(api),
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
	j.client.Log.Debug("Legal Hold daily job scheduled")

	j.onceScheduler.SetCallback(j.runOnce)
	if err := j.onceScheduler.Start(); err != nil {
		return fmt.Errorf("could not start scheduler for runOnce jobs: %w", err)
	}
	j.client.Log.Debug("Legal Hold runOnce cluster scheduler started")

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

func (j *LegalHoldJob) RunAll() {
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

	for _, lh := range []model.LegalHold{*legalHold} {
		legalHold := lh.DeepCopy()

		j.client.Log.Info("Creating legal hold ad-hoc job", "legal_hold_id", legalHold.ID)

		j.onceScheduler.ScheduleOnce(
			"legal_hold_run_"+lh.ID,
			time.Now(),
			&LegalHoldRunOnceProps{
				LegalHold: legalHold,
				ForceRun:  true,
			},
		)
	}

	// j.runWith([]model.LegalHold{*legalHold}, true)
	return nil
}

func (j *LegalHoldJob) GetRunningLegalHolds() ([]string, error) {
	jobs, err := j.onceScheduler.ListScheduledJobs()
	if err != nil {
		return nil, fmt.Errorf("failed to list scheduled jobs: %w", err)
	}

	var runningJobs []string

	for _, job := range jobs {
		if strings.HasPrefix(job.Key, "legal_hold_run_") {
			runningJobs = append(runningJobs, job.Key)
		}
	}

	return runningJobs, nil
}

func (j *LegalHoldJob) runOnce(key string, props any) {
	runOnceProps, ok := props.(*LegalHoldRunOnceProps)
	if !ok {
		j.client.Log.Error("LegalHoldJob: invalid run once props")
		return
	}

	time.Sleep(time.Minute * 2)

	j.runWith(
		[]model.LegalHold{runOnceProps.LegalHold},
		runOnceProps.ForceRun,
	)
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
		if lh.IsFinished() {
			j.client.Log.Debug(fmt.Sprintf("Legal Hold %s has ended and therefore does not executing.", lh.ID))
			continue
		}

		now := mattermostModel.GetMillis()
		if !forceRun && !lh.NeedsExecuting(now) {
			j.client.Log.Debug(fmt.Sprintf("Legal Hold %s is not yet ready to be executed again.", lh.ID))
			continue
		}
		if !forceRun && lh.LastExecutionEndedAt >= now {
			j.client.Log.Debug(fmt.Sprintf("Legal Hold %s was already executed after the current time.", lh.ID))
			continue
		}

		j.client.Log.Debug(fmt.Sprintf("Creating Legal Hold Execution for legal hold: %s", lh.ID))
		lhe := legalhold.NewExecution(lh, j.papi, j.sqlstore, j.kvstore, j.filebackend)

		if updatedLH, err := lhe.Execute(); err != nil {
			if strings.Contains(err.Error(), "another execution is already running") {
				j.client.Log.Debug("Another execution is already running for this legal hold", "legal_hold_id", lh.ID)
				continue
			}
			j.client.Log.Error("An error occurred executing the legal hold.", err)
		} else {
			// Update legal hold with the new execution details (last execution time and last message)
			// Also set it to IDLE again since the execution has ended.
			old, err := j.kvstore.GetLegalHoldByID(lh.ID)
			if err != nil {
				j.client.Log.Error("Failed to fetch the LegalHold prior to updating", err)
				continue
			}
			lh = *old
			lh.LastExecutionEndedAt = updatedLH.LastExecutionEndedAt
			lh.HasMessages = true
			lh.Status = model.LegalHoldStatusIdle
			newLH, err := j.kvstore.UpdateLegalHold(lh, *old)
			if err != nil {
				j.client.Log.Error("Failed to update legal hold", err)
				continue
			}
			lh = *newLH
			j.client.Log.Info("legal hold executed", "legal_hold_id", lh.ID, "legal_hold_name", lh.Name)
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
