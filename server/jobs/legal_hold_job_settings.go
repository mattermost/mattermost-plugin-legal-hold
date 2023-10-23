package jobs

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/config"
)

const (
	FullLayout      = "Jan 2, 2006 3:04pm -0700"
	TimeOfDayLayout = "3:04pm -0700"
)

type LegalHoldJobSettings struct {
	EnableLegalHoldJobs bool
}

func (s *LegalHoldJobSettings) Clone() *LegalHoldJobSettings {
	return &LegalHoldJobSettings{
		EnableLegalHoldJobs: s.EnableLegalHoldJobs,
	}
}

func (s *LegalHoldJobSettings) String() string {
	return fmt.Sprintf("enabled=%T", s.EnableLegalHoldJobs)
}

func parseLegaHoldJobSettings(cfg *config.Configuration) (*LegalHoldJobSettings, error) {
	if cfg == nil {
		return &LegalHoldJobSettings{
			EnableLegalHoldJobs: false,
		}, nil
	}

	return &LegalHoldJobSettings{
		EnableLegalHoldJobs: true,
	}, nil
}

func (s *LegalHoldJobSettings) CalcNext(last time.Time) time.Time {
	// TODO: Make this configurable. Channel Archiver provides example code for how to do this.
	timeOfDay := time.Date(2009, 11, 17, 8, 16, 0, 0, time.UTC)
	// return time.Date(last.Year(), last.Month(), last.Day()+1, timeOfDay.Hour(), timeOfDay.Minute(), timeOfDay.Second(), 0, timeOfDay.Location())
	return time.Date(last.Year(), last.Month(), last.Day(), last.Hour(), last.Minute()+2, timeOfDay.Second(), 0, last.Location())
}
