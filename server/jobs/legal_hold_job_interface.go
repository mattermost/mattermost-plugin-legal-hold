package jobs

import (
	"time"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
)

// LegalHoldJobInterface defines the interface that both real and mock implementations must satisfy
type LegalHoldJobInterface interface {
	GetID() string
	OnConfigurationChange(cfg *config.Configuration) error
	Stop(timeout time.Duration) error
	RunFromAPI()
	RunSingleLegalHold(id string) error
}
