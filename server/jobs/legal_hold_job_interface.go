package jobs

import (
	"time"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
)

// LegalHoldJobInterface defines the interface that both real and mock implementations must satisfy
type LegalHoldJobInterface interface {
	GetID() string
	GetRunningLegalHolds() (ids []string, err error)
	OnConfigurationChange(cfg *config.Configuration) error
	Stop(timeout time.Duration) error
	RunAll()
	RunSingleLegalHold(id string) error
}
