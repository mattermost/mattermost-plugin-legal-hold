package main

import (
	"fmt"
	"net/http"
	"sync"

	pluginapi "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/plugin"

	"github.com/mattermost/mattermost-plugin-legalhold/server/store"
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
	configuration *configuration

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
