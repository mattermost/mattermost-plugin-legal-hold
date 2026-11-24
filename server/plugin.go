package main

import (
	"fmt"
	"os"
	"path/filepath"
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

	// FileBackend is where legal hold data is written.
	FileBackend filestore.FileBackend

	// SourceFileBackend is the Mattermost file store. Attachments are read from
	// here. It equals FileBackend when the hold uses the server store.
	SourceFileBackend filestore.FileBackend

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
	// A failure leaves the saved selection unchanged. Switching stores on a
	// transient error would send later reads and writes to a different place
	// than the archives already written.
	filesBackend, err := p.openFileBackend()
	if err != nil {
		p.Client.Log.Error("unable to initialize the files storage", "err", err)
		return errors.Wrap(err, "unable to initialize the files storage")
	}

	sourceBackend := filesBackend
	if p.getConfiguration().AmazonS3BucketSettings.EffectiveDriver() != config.FilestoreDriverServer {
		if p.Client.Configuration.GetUnsanitizedConfig() == nil {
			return errors.New("server configuration is not available")
		}
		sourceBackend, err = p.newFileBackend(p.Client.Configuration.GetUnsanitizedConfig().FileSettings)
		if err != nil {
			p.Client.Log.Error("unable to initialize the Mattermost file store", "err", err)
			return errors.Wrap(err, "unable to initialize the Mattermost file store")
		}
	}

	// Perform a filesystem check if enabled
	if p.configuration.EnableFilestoreConnectionTest {
		p.Client.Log.Debug("Performing filestore connection test")
		if err = filesBackend.TestConnection(); err != nil {
			err = errors.Wrap(err, "connection test for filestore failed")
			p.Client.Log.Error(err.Error())
			return err
		}
		if sourceBackend != filesBackend {
			if err = sourceBackend.TestConnection(); err != nil {
				err = errors.Wrap(err, "connection test for the Mattermost file store failed")
				p.Client.Log.Error(err.Error())
				return err
			}
		}
	} else {
		p.Client.Log.Debug("Skipping filestore connection test")
	}

	p.FileBackend = filesBackend
	p.SourceFileBackend = sourceBackend

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
	p.legalHoldJob, err = jobs.NewLegalHoldJob(LegalHoldJobID, p.API, p.Client, p.SQLStore, p.KVStore, p.FileBackend, p.SourceFileBackend)
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
		p.Client.Log.Debug("Legal hold filestore is a dedicated Azure Blob container", "account", settings.StorageAccount, "container", settings.Container, "auth_mode", settings.AuthMode)
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
	settings := azureblob.Settings{
		StorageAccount: in.StorageAccount,
		AuthMode:       in.AuthMode,
		AccessKey:      in.AccessKey,
		Container:      in.Container,
		PathPrefix:     in.PathPrefix,
		Cloud:          in.Cloud,
		Endpoint:       in.Endpoint,
		SSL:            in.SSL,
	}
	if in.RequestTimeoutMilliseconds > 0 {
		timeout := in.RequestTimeoutMilliseconds
		settings.RequestTimeoutMilliseconds = &timeout
	}
	return settings
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
		p.Client.Log.Debug("Legal hold filestore is Azure Blob", "account", settings.StorageAccount, "container", settings.Container, "auth_mode", settings.AuthMode)
		return azureblob.New(settings)
	}
	return filestore.NewFileBackend(FixedFileSettingsToFileBackendSettings(fileSettings))
}

func (p *Plugin) azureBlobSettings() (azureblob.Settings, error) {
	// MM_CONFIG names the live server config. A file that cannot be read, or a
	// database query that fails, is returned as an error. A different file is
	// not substituted. Environment variables still override afterwards, which
	// is how an env-only deployment supplies Azure settings.
	mmConfig := os.Getenv("MM_CONFIG")
	raw, err := loadServerConfigJSON(mmConfig, os.ReadFile, func() ([]byte, error) {
		if strings.Contains(mmConfig, "://") {
			return readActiveConfigurationDSN(mmConfig)
		}
		if p.SQLStore == nil {
			return nil, nil
		}
		return p.SQLStore.ActiveConfiguration()
	})
	// A named file or DSN that fails is not replaced with some other source.
	// An empty MM_CONFIG uses the server's default file search (config/config.json,
	// then config.json). A missing file is env-only. The application database is
	// not consulted: after a rollback off database config, its row is stale.
	if err != nil {
		return azureblob.Settings{}, err
	}
	settings, err := azureblob.SettingsFromConfigJSON(raw)
	if err != nil {
		return azureblob.Settings{}, err
	}
	return azureblob.ApplyEnv(settings, os.LookupEnv), nil
}

// defaultServerConfigFile is the file name the server uses when --config and
// MM_CONFIG are both unset. The server looks for it under config/ before the
// working directory.
const defaultServerConfigFile = "config.json"

// loadServerConfigJSON reads the server config document from the source
// MM_CONFIG selects. An empty MM_CONFIG uses the same relative search as the
// server. A missing default file is env-only, not an error and not a database
// read. A named file or DSN that fails is returned as-is.
func loadServerConfigJSON(mmConfig string, readFile func(string) ([]byte, error), readDB func() ([]byte, error)) ([]byte, error) {
	if mmConfig == "" {
		return readDefaultServerConfig(readFile)
	}
	if strings.Contains(mmConfig, "://") {
		if readDB == nil {
			return nil, errors.New("MM_CONFIG names a database but no config reader is available")
		}
		return readDB()
	}
	if filepath.IsAbs(mmConfig) {
		raw, err := readFile(mmConfig)
		if err != nil {
			return nil, fmt.Errorf("read MM_CONFIG file: %w", err)
		}
		return raw, nil
	}
	return readNamedServerConfig(mmConfig, readFile)
}

// readDefaultServerConfig follows the server's search for an unset MM_CONFIG:
// config/config.json near the working directory or the executable, then
// config.json itself. The first readable file wins. A path that exists but
// cannot be read is returned; later candidates are not a fallback.
func readDefaultServerConfig(readFile func(string) ([]byte, error)) ([]byte, error) {
	if readFile == nil {
		return nil, nil
	}
	for _, candidate := range configCandidates(defaultServerConfigFile) {
		raw, err := readFile(candidate)
		if err == nil {
			return raw, nil
		}
		if os.IsNotExist(err) {
			continue
		}
		return nil, fmt.Errorf("read default config file %s: %w", candidate, err)
	}
	return nil, nil
}

// readNamedServerConfig resolves a relative MM_CONFIG name the way the server
// does. An explicit name that cannot be found is an error. A file that exists
// but cannot be read is not replaced with a later candidate.
func readNamedServerConfig(name string, readFile func(string) ([]byte, error)) ([]byte, error) {
	if readFile == nil {
		return nil, fmt.Errorf("read MM_CONFIG file %s: %w", name, os.ErrNotExist)
	}
	for _, candidate := range configCandidates(name) {
		raw, err := readFile(candidate)
		if err == nil {
			return raw, nil
		}
		if os.IsNotExist(err) {
			continue
		}
		return nil, fmt.Errorf("read MM_CONFIG file %s: %w", candidate, err)
	}
	return nil, fmt.Errorf("read MM_CONFIG file %s: %w", name, os.ErrNotExist)
}

// configCandidates matches mattermost-server config.resolveConfigFilePath for a
// relative file name: config/<name> walked up from the working directory and
// the executable, then the name in the working directory.
func configCandidates(name string) []string {
	var candidates []string
	seen := map[string]struct{}{}
	add := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		candidates = append(candidates, p)
	}
	var roots []string
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	for _, root := range roots {
		dir := root
		for range 8 {
			add(filepath.Join(dir, "config", name))
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	add(name)
	if wd, err := os.Getwd(); err == nil {
		add(filepath.Join(wd, name))
	}
	return candidates
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
