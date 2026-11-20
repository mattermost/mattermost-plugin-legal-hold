package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	pluginapi "github.com/mattermost/mattermost-plugin-api"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/azureblob"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/jobs"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/kvstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
)

const (
	LegalHoldJobID              = "legal_hold_job"
	MattermostEntrySkuShortName = "entry"
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
	legalHoldJob jobs.LegalHoldJobInterface

	// router holds the HTTP router for the plugin's rest API
	router *mux.Router
}

func (p *Plugin) OnActivate() error {
	// Check for an enterprise license or a development environment
	config := p.API.GetConfig()
	license := p.API.GetLicense()

	if !pluginapi.IsEnterpriseLicensedOrDevelopment(config, license) || (!pluginapi.IsConfiguredForDevelopment(config) && license.SkuShortName == MattermostEntrySkuShortName) {
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
	configuration := new(config.Configuration)

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

	// Reinitialise the filestore backend. A dedicated S3 bucket, Azure Blob
	// container, or local directory is used when selected. Otherwise the
	// server file store is used, including its own azureblob driver.
	filesBackend, err := p.openFileBackend()
	if err != nil {
		if fbErr := p.fallbackDedicatedFilestore(); fbErr != nil {
			return fbErr
		}
		p.Client.Log.Error("unable to initialize the files storage", "err", err)
		return errors.Wrap(err, "unable to initialize the files storage")
	}

	// Perform a filesystem check if enabled
	if p.configuration.EnableFilestoreConnectionTest {
		p.Client.Log.Debug("Performing filestore connection test")
		if err = filesBackend.TestConnection(); err != nil {
			// A bad dedicated store must not leave the plugin dead, or the
			// admin cannot open this page to correct it. Fall back to the
			// server file store. A failure of that server store is left as-is.
			if fbErr := p.fallbackDedicatedFilestore(); fbErr != nil {
				return fbErr
			}
			err = errors.Wrap(err, "connection test for filestore failed")
			p.Client.Log.Error(err.Error())
			return err
		}
	} else {
		p.Client.Log.Debug("Skipping filestore connection test")
	}

	p.FileBackend = filesBackend

	// Check all legal holds on plugin activation to prevent corrupt states:
	// - For legal holds that supposedly don't have messages, check if the index exist
	//   and update the field accordingly.
	//
	// Ignore errors during plugin activation to ensure working operation and
	// allowing debugging in parallel since most errors on this block would come
	// from KVStorage problems and that would require manual intervention in the
	// database.
	legalHolds, err := p.KVStore.GetAllLegalHolds()
	if err != nil {
		p.Client.Log.Error("Failed to get legal holds during activation", "err", err)
		return err
	}

	for _, lh := range legalHolds {
		oldLH := lh.DeepCopy()

		// Check the filebackend if the legal hold doesn't have messages, continue otherwise.
		if !lh.HasMessages {
			// Update the field value
			lh.HasMessages, err = p.FileBackend.FileExists(lh.IndexPath())
			if err != nil {
				p.Client.Log.Error("Failed to check if legal hold index exists during activation", "legal_hold_id", lh.ID, "err", err)
			}
		}

		// Update the legal hold
		if _, err = p.KVStore.UpdateLegalHold(lh, oldLH); err != nil {
			p.Client.Log.Error("Failed to update legal hold during activation", "legal_hold_id", lh.ID, "err", err)
		}
	}

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

// openFileBackend opens the store selected in the plugin settings, or the
// Mattermost server file store when the selection is "server".
func (p *Plugin) openFileBackend() (filestore.FileBackend, error) {
	conf := p.getConfiguration()
	switch conf.AmazonS3BucketSettings.EffectiveDriver() {
	case config.FilestoreDriverS3:
		settings := conf.AmazonS3BucketSettings.Settings
		if settings.DriverName == nil || *settings.DriverName == "" {
			settings.DriverName = mattermostModel.NewString(mattermostModel.ImageDriverS3)
		}
		return filestore.NewFileBackend(FixedFileSettingsToFileBackendSettings(settings))
	case config.FilestoreDriverAzure:
		settings := pluginAzureSettings(conf.AmazonS3BucketSettings.Azure)
		p.Client.Log.Info("Legal hold filestore is a dedicated Azure Blob container", "account", settings.StorageAccount, "container", settings.Container, "auth_mode", settings.AuthMode)
		return azureblob.New(settings)
	case config.FilestoreDriverLocal:
		if conf.AmazonS3BucketSettings.Directory == "" {
			return nil, errors.New("local filestore directory is required")
		}
		return filestore.NewFileBackend(filestore.FileBackendSettings{
			DriverName: mattermostModel.ImageDriverLocal,
			Directory:  conf.AmazonS3BucketSettings.Directory,
		})
	default:
		if p.Client.Configuration.GetUnsanitizedConfig() == nil {
			return nil, errors.New("server configuration is not available")
		}
		return p.newFileBackend(p.Client.Configuration.GetUnsanitizedConfig().FileSettings)
	}
}

func pluginAzureSettings(in config.AzureBlobSettings) azureblob.Settings {
	ssl := in.SSL
	settings := azureblob.Settings{
		StorageAccount: in.StorageAccount,
		AuthMode:       in.AuthMode,
		AccessKey:      in.AccessKey,
		Container:      in.Container,
		PathPrefix:     in.PathPrefix,
		Cloud:          in.Cloud,
		Endpoint:       in.Endpoint,
		SSL:            &ssl,
	}
	if in.RequestTimeoutMilliseconds > 0 {
		timeout := in.RequestTimeoutMilliseconds
		settings.RequestTimeoutMilliseconds = &timeout
	}
	return settings
}

// fallbackDedicatedFilestore points a failing dedicated store back at the
// server file store so the plugin can start and the admin can edit the form.
// The bucket, container, and directory values are left in the saved config.
func (p *Plugin) fallbackDedicatedFilestore() error {
	if p.getConfiguration().AmazonS3BucketSettings.EffectiveDriver() == config.FilestoreDriverServer {
		return nil
	}
	pluginConfig := p.Client.Configuration.GetPluginConfig()
	settings, ok := pluginConfig["amazons3bucketsettings"].(map[string]any)
	if !ok || settings == nil {
		p.Client.Log.Warn("Filestore settings not found in plugin config. Could not fall back to the server file store.")
		return nil
	}
	settings["Enable"] = false
	settings["DriverName"] = config.FilestoreDriverServer
	if err := p.Client.Configuration.SavePluginConfig(pluginConfig); err != nil {
		p.Client.Log.Error("Error while saving plugin config.", "Error", err.Error())
		return err
	}
	return nil
}

// newFileBackend opens the Mattermost server file store.
// DriverName azureblob cannot go through the v6 filestore: that package
// rejects the driver, and the v6 FileSettings struct drops Azure* fields
// when the server config is decoded. Those fields are read from the
// configurations table, MM_CONFIG, and MM_FILESETTINGS_AZURE*.
func (p *Plugin) newFileBackend(fileSettings mattermostModel.FileSettings) (filestore.FileBackend, error) {
	fileSettings.SetDefaults(true)
	if fileSettings.DriverName != nil && *fileSettings.DriverName == azureblob.DriverName {
		settings, err := p.azureBlobSettings()
		if err != nil {
			return nil, err
		}
		p.Client.Log.Info("Legal hold filestore is Azure Blob", "account", settings.StorageAccount, "container", settings.Container, "auth_mode", settings.AuthMode)
		return azureblob.New(settings)
	}
	return filestore.NewFileBackend(FixedFileSettingsToFileBackendSettings(fileSettings))
}

func (p *Plugin) azureBlobSettings() (azureblob.Settings, error) {
	// MM_CONFIG decides which store is actually live. A file path wins over a
	// leftover configurations row; a DSN means the table is the store. Env
	// vars are applied last because the server gives them the same priority.
	var raw []byte
	configPath := os.Getenv("MM_CONFIG")
	switch {
	case configPath != "" && !strings.Contains(configPath, "://"):
		raw = readFileIfExists(configPath)
	case p.SQLStore != nil:
		stored, err := p.SQLStore.ActiveConfiguration()
		if err != nil {
			p.Client.Log.Debug("No active configuration row for Azure file settings", "err", err)
		} else {
			raw = stored
		}
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		for _, candidate := range []string{"config/config.json", "config.json"} {
			if b := readFileIfExists(candidate); len(b) > 0 {
				raw = b
				break
			}
		}
	}
	settings, err := azureblob.SettingsFromConfigJSON(raw)
	if err != nil {
		return azureblob.Settings{}, err
	}
	return azureblob.ApplyEnv(settings, os.LookupEnv), nil
}

func readFileIfExists(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

// FixedFileSettingsToFileBackendSettings converts FileSettings to FileBackendSettings.
// This is a copy of FileSettings.ToFileBackendSettings from mattermost-server with nil-safe
// handling for boolean pointer fields, as the original method may be removed in future versions.
// This function internally calls SetDefaults(true) to ensure all pointer fields are initialized.
func FixedFileSettingsToFileBackendSettings(fileSettings mattermostModel.FileSettings) filestore.FileBackendSettings {
	// SetDefaults(true) initializes nil pointer fields with sensible defaults while preserving
	// any existing user-configured values. This prevents nil pointer dereferences when accessing
	// string and int64 pointer fields like AmazonS3Bucket, AmazonS3Region, etc.
	fileSettings.SetDefaults(true)

	if *fileSettings.DriverName == mattermostModel.ImageDriverLocal {
		return filestore.FileBackendSettings{
			DriverName: *fileSettings.DriverName,
			Directory:  *fileSettings.Directory,
		}
	}

	return filestore.FileBackendSettings{
		DriverName:                         *fileSettings.DriverName,
		AmazonS3AccessKeyId:                *fileSettings.AmazonS3AccessKeyId,
		AmazonS3SecretAccessKey:            *fileSettings.AmazonS3SecretAccessKey,
		AmazonS3Bucket:                     *fileSettings.AmazonS3Bucket,
		AmazonS3PathPrefix:                 *fileSettings.AmazonS3PathPrefix,
		AmazonS3Region:                     *fileSettings.AmazonS3Region,
		AmazonS3Endpoint:                   *fileSettings.AmazonS3Endpoint,
		AmazonS3SSL:                        fileSettings.AmazonS3SSL == nil || *fileSettings.AmazonS3SSL,
		AmazonS3SignV2:                     fileSettings.AmazonS3SignV2 != nil && *fileSettings.AmazonS3SignV2,
		AmazonS3SSE:                        fileSettings.AmazonS3SSE != nil && *fileSettings.AmazonS3SSE,
		AmazonS3Trace:                      fileSettings.AmazonS3Trace != nil && *fileSettings.AmazonS3Trace,
		AmazonS3RequestTimeoutMilliseconds: *fileSettings.AmazonS3RequestTimeoutMilliseconds,
		SkipVerify:                         false,
	}
}
