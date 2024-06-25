package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/jobs"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/kvstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
)

const (
	LegalHoldJobID    = "legal_hold_job"
	LegalHoldPluginID = "com.mattermost.plugin-legal-hold"
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
	// Check for an enterprise license or a development environment
	config := p.API.GetConfig()
	license := p.API.GetLicense()

	if !pluginapi.IsEnterpriseLicensedOrDevelopment(config, license) {
		return fmt.Errorf("this plugin requires an Enterprise license")
	}

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

func (p *Plugin) ConfigurationWillBeSaved(newCfg *model.Config) (*model.Config, error) {
	oldPluginConf := p.getConfiguration()

	newPluginSettings := newCfg.PluginSettings.Plugins[LegalHoldPluginID]
	if newPluginSettings == nil {
		return newCfg, nil
	}

	newPluginSettingsBytes, err := json.Marshal(newPluginSettings)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal plugin settings")
	}

	newPluginConf := &config.Configuration{}
	if err := json.Unmarshal(newPluginSettingsBytes, newPluginConf); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal plugin settings")
	}

	if newPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey != nil &&
		*newPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey != "" &&
		*newPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey != model.FakeSetting &&
		oldPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey != nil &&
		*newPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey != *oldPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey {

		newSecret := *newPluginConf.AmazonS3BucketSettings.Settings.AmazonS3SecretAccessKey

		s3Settings := newCfg.PluginSettings.Plugins[LegalHoldPluginID]["amazons3bucketsettings"]
		s3SettingsMap, ok := s3Settings.(map[string]interface{})
		if !ok {
			return nil, errors.New("failed to cast s3Settings to map[string]interface{}")
		}

		actualSettings, ok := s3SettingsMap["Settings"].(map[string]interface{})
		if !ok {
			return nil, errors.New("failed to cast actualSettings to map[string]interface{}")
		}

		actualSettings["AmazonS3SecretAccessKey"] = model.FakeSetting
		err = p.saveS3Secret(newSecret)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save s3 secret")
		}
	}

	return newCfg, nil
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

	conf := p.getConfiguration()

	serverFileSettings := p.Client.Configuration.GetUnsanitizedConfig().FileSettings
	if conf.AmazonS3BucketSettings.Enable {
		serverFileSettings = conf.AmazonS3BucketSettings.Settings

		s3Secret, err := p.getS3Secret()
		if err != nil {
			return err
		}

		if len(s3Secret) > 0 {
			serverFileSettings.AmazonS3SecretAccessKey = model.NewString(string(s3Secret))
		}
	}

	// Reinitialise the filestore backend
	// FIXME: Boolean flags shouldn't be hard coded.
	filesBackendSettings := FixedFileSettingsToFileBackendSettings(serverFileSettings, false, true)
	filesBackend, err := filestore.NewFileBackend(filesBackendSettings)
	if err != nil {
		p.Client.Log.Error("unable to initialize the files storage", "err", err)
		return errors.New("unable to initialize the files storage")
	}

	if err = filesBackend.TestConnection(); err != nil {
		err = errors.Wrap(err, "connection test for filestore failed")
		p.Client.Log.Error(err.Error())
		return err
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

func FixedFileSettingsToFileBackendSettings(fileSettings model.FileSettings, enableComplianceFeature bool, skipVerify bool) filestore.FileBackendSettings {
	if *fileSettings.DriverName == model.ImageDriverLocal {
		return filestore.FileBackendSettings{
			DriverName: *fileSettings.DriverName,
			Directory:  *fileSettings.Directory,
		}
	}

	amazonS3Bucket := ""
	if fileSettings.AmazonS3Bucket != nil {
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

func (p *Plugin) saveS3Secret(secret string) error {
	return p.KVStore.SetAWSSecretKey(secret)
}

func (p *Plugin) getS3Secret() (string, error) {
	return p.KVStore.GetAWSSecretKey()
}
