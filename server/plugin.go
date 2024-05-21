package main

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/jobs"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/kvstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
)

const (
	LegalHoldJobID = "legal_hold_job"
)

// Plugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *config.Configuration

	// Client is the client interface for the Mattermost plugin API
	Client *pluginapi.Client

	// SQLStore allows direct access to the Mattermost store bypassing the plugin API
	SQLStore *sqlstore.SQLStore

	// KVStore provides tailored access to this plugin's KV Store.
	KVStore kvstore.KVStore

	// FileBackend allows direct access to the Mattermost files backend bypassing the plugin API.
	FileBackend filestore.FileBackend

	// jobManager allows managing of scheduled tasks
	jobManager *jobs.JobManager

	// legalHoldJob runs the legal hold jobs
	legalHoldJob *jobs.LegalHoldJob

	// router holds the HTTP router for the plugin's rest API
	router *mux.Router
}

func (p *Plugin) OnActivate() error {
	// Create plugin API client
	p.Client = pluginapi.NewClient(p.API, p.Driver)
	p.Client.Log.Debug("MM LH Plugin: OnActivate called")

	err := p.Client.KV.Delete("cron_legal_hold_job")
	if err != nil {
		return err
	}

	// Setup direct store access
	SQLStore, err := sqlstore.New(p.Client.Store, &p.Client.Log)
	if err != nil {
		p.Client.Log.Error("cannot create SQLStore", "err", err)
		return err
	}
	p.SQLStore = SQLStore
	// FIXME: do we need to handle MM configuration changes?

	p.KVStore = kvstore.NewKVStore(p.Client)

	// Create job manager
	p.jobManager = jobs.NewJobManager(&p.Client.Log)

	return p.Reconfigure()
}

// getConfiguration retrieves the active Configuration under lock, making it safe to use
// concurrently. The active Configuration may change underneath the client of this method, but
// the struct returned by this API call is considered immutable.
func (p *Plugin) getConfiguration() *config.Configuration {
	p.configurationLock.RLock()
	defer p.configurationLock.RUnlock()

	if p.configuration == nil {
		return &config.Configuration{}
	}

	return p.configuration
}

// setConfiguration replaces the active Configuration under lock.
//
// Do not call setConfiguration while holding the configurationLock, as sync.Mutex is not
// reentrant. In particular, avoid using the plugin API entirely, as this may in turn trigger a
// hook back into the plugin. If that hook attempts to acquire this lock, a deadlock may occur.
//
// This method panics if setConfiguration is called with the existing Configuration. This almost
// certainly means that the Configuration was modified without being cloned and may result in
// an unsafe access.
func (p *Plugin) setConfiguration(configuration *config.Configuration) {
	p.configurationLock.Lock()
	defer p.configurationLock.Unlock()

	if configuration != nil && p.configuration == configuration {
		// Ignore assignment if the Configuration struct is empty. Go will optimize the
		// allocation for same to point at the same memory address, breaking the check
		// above.
		if reflect.ValueOf(*configuration).NumField() == 0 {
			return
		}

		panic("setConfiguration called with the existing Configuration")
	}

	p.configuration = configuration
}

// OnConfigurationChange is invoked when Configuration changes may have been made.
func (p *Plugin) OnConfigurationChange() error {
	var configuration = new(config.Configuration)

	// Load the public Configuration fields from the Mattermost server Configuration.
	if err := p.API.LoadPluginConfiguration(configuration); err != nil {
		return errors.Wrap(err, "failed to load plugin Configuration")
	}

	p.setConfiguration(configuration)

	return p.Reconfigure()
}

func (p *Plugin) Reconfigure() error {
	// Don't do anything if the plugin isn't activated yet.
	if p.Client == nil {
		return nil
	}

	p.Client.Log.Debug("Plugin.Reconfigure() called")

	if p.Client.Configuration.GetConfig() == nil {
		p.Client.Log.Info("Client GetConfig() is nil")
		return nil
	}

	pluginConfig := p.getConfiguration()
	customS3Bucket := pluginConfig.S3Bucket

	// Reinitialise the filestore backend
	// FIXME: Boolean flags shouldn't be hard coded.
	filesBackendSettings := FixedFileSettingsToFileBackendSettings(p.Client.Configuration.GetUnsanitizedConfig().FileSettings, customS3Bucket, false, true)
	filesBackend, err := filestore.NewFileBackend(filesBackendSettings)
	if err != nil {
		p.Client.Log.Error("unable to initialize the files storage", "err", err)
		return errors.New("unable to initialize the files storage")
	}
	p.FileBackend = filesBackend

	// Remove old job if exists
	if p.legalHoldJob != nil {
		if err = p.jobManager.RemoveJob(LegalHoldJobID, 0); err != nil {
			return err
		}
		p.Client.Log.Info("Stopped old job")
	}

	// Create new job
	p.legalHoldJob, err = jobs.NewLegalHoldJob(LegalHoldJobID, p.API, p.Client, p.SQLStore, p.KVStore, p.FileBackend)
	if err != nil {
		return fmt.Errorf("cannot create legal hold job: %w", err)
	}
	if err := p.jobManager.AddJob(p.legalHoldJob); err != nil {
		return fmt.Errorf("cannot add legal hold job: %w", err)
	}
	_ = p.jobManager.OnConfigurationChange(p.getConfiguration())

	return nil
}

func FixedFileSettingsToFileBackendSettings(fileSettings model.FileSettings, customS3Bucket string, enableComplianceFeature bool, skipVerify bool) filestore.FileBackendSettings {
	if *fileSettings.DriverName == model.ImageDriverLocal {
		return filestore.FileBackendSettings{
			DriverName: *fileSettings.DriverName,
			Directory:  *fileSettings.Directory,
		}
	}

	amazonS3Bucket := ""
	if customS3Bucket != "" {
		amazonS3Bucket = customS3Bucket
	} else if fileSettings.AmazonS3Bucket != nil {
		amazonS3Bucket = *fileSettings.AmazonS3Bucket
	}

	amazonS3PathPrefix := ""
	if fileSettings.AmazonS3PathPrefix != nil {
		amazonS3PathPrefix = *fileSettings.AmazonS3PathPrefix
	}

	amazonS3Region := ""
	if fileSettings.AmazonS3Region != nil {
		amazonS3Region = *fileSettings.AmazonS3Region
	}

	return filestore.FileBackendSettings{
		DriverName:                         *fileSettings.DriverName,
		AmazonS3AccessKeyId:                *fileSettings.AmazonS3AccessKeyId,
		AmazonS3SecretAccessKey:            *fileSettings.AmazonS3SecretAccessKey,
		AmazonS3Bucket:                     amazonS3Bucket,
		AmazonS3PathPrefix:                 amazonS3PathPrefix,
		AmazonS3Region:                     amazonS3Region,
		AmazonS3Endpoint:                   *fileSettings.AmazonS3Endpoint,
		AmazonS3SSL:                        fileSettings.AmazonS3SSL != nil && *fileSettings.AmazonS3SSL,
		AmazonS3SignV2:                     fileSettings.AmazonS3SignV2 != nil && *fileSettings.AmazonS3SignV2,
		AmazonS3SSE:                        fileSettings.AmazonS3SSE != nil && *fileSettings.AmazonS3SSE && enableComplianceFeature,
		AmazonS3Trace:                      fileSettings.AmazonS3Trace != nil && *fileSettings.AmazonS3Trace,
		AmazonS3RequestTimeoutMilliseconds: *fileSettings.AmazonS3RequestTimeoutMilliseconds,
		SkipVerify:                         skipVerify,
	}
}
