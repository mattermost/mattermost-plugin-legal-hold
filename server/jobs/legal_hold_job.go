package jobs

import (
	"context"
	"fmt"
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
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store"
)

type LegalHoldJob struct {
	mux      sync.Mutex
	settings *LegalHoldJobSettings
	job      *cluster.Job
	runner   *runInstance

	id          string
	papi        plugin.API
	client      *pluginapi.Client
	sqlstore    *store.SQLStore
	filebackend filestore.FileBackend
}

func NewLegalHoldJob(id string, api plugin.API, client *pluginapi.Client, sqlstore *store.SQLStore, filebackend filestore.FileBackend) (*LegalHoldJob, error) {
	return &LegalHoldJob{
		settings:    &LegalHoldJobSettings{},
		id:          id,
		papi:        api,
		client:      client,
		sqlstore:    sqlstore,
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

	next := settings.CalcNext(lastFinished)
	delta := next.Sub(now)

	j.client.Log.Debug("Legal Hold Job next run scheduled", "last", lastFinished.Format(FullLayout), "next", next.Format(FullLayout), "wait", delta.String())

	return delta
}

func (j *LegalHoldJob) run() {
	j.client.Log.Info("Running Legal Hold Job")
	exitSignal := make(chan struct{})
	ctx, canceller := context.WithCancel(context.Background())

	runner := &runInstance{
		canceller:  canceller,
		exitSignal: exitSignal,
	}

	var oldRunner *runInstance
	var settings *LegalHoldJobSettings
	j.mux.Lock()
	oldRunner = j.runner
	j.runner = runner
	settings = j.settings.Clone()
	j.mux.Unlock()

	defer func() {
		close(exitSignal)
		j.mux.Lock()
		j.runner = nil
		j.mux.Unlock()
	}()

	if oldRunner != nil {
		j.client.Log.Error("Multiple Legal Hold jobs scheduled concurrently; there can be only one")
		return
	}

	// FIXME: Legal Holds need to be retrieved from the KV Store, not hard coded.
	lh := model.LegalHold{
		ID:                   mattermostModel.NewId(),
		UserIDs:              []string{"pou3tf4q97nt5rthqdcaf5cxwr"},
		StartsAt:             mattermostModel.GetMillis() - 86400000/2,
		EndsAt:               mattermostModel.GetMillis() + 86400000/2,
		LastExecutionEndedAt: mattermostModel.GetMillis() - 86400000/2,
		ExecutionLength:      864000,
	}
	lhe := legalhold.NewExecution(lh, j.sqlstore, j.filebackend)
	if err := lhe.Execute(); err != nil {
		j.client.Log.Error("An error occurred executing the legal hold.", err)
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
