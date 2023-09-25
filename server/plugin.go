package main

import (
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"reflect"
	"sync"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store"
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
	SQLStore *store.SQLStore

	// jobManager allows managing of scheduled tasks
	//jobManager *jobs.JobManager
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	// TODO: Remove hello world code.
	fmt.Fprint(w, "Hello, world!")
}

func (p *Plugin) OnActivate() error {
	// Create plugin API client
	p.Client = pluginapi.NewClient(p.API, p.Driver)

	// Setup direct store access
	SQLStore, err := store.New(p.Client.Store, &p.Client.Log)
	if err != nil {
		p.Client.Log.Error("cannot create SQLStore", "err", err)
		return err
	}
	p.SQLStore = SQLStore

	// Create job manager
	//p.jobManager = jobs.NewJobManager(&p.Client.Log)

	// Create job for legal hold execution
	// FIXME: Implement me!
	/*
		channelArchiverJob, err := jobs.NewChannelArchiverJob(LegalHoldJobID, p.API, p.Client, SQLStore)
		if err != nil {
			return fmt.Errorf("cannot create channel archiver job: %w", err)
		}
		if err := p.jobManager.AddJob(channelArchiverJob); err != nil {
			return fmt.Errorf("cannot add channel archiver job: %w", err)
		}
		_ = p.jobManager.OnConfigurationChange(p.getConfiguration())
	*/

	return nil
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

	return nil
}
