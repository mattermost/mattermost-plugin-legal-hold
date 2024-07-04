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
	TimeOfDay           time.Time
}

func (s *LegalHoldJobSettings) Clone() *LegalHoldJobSettings {
	return &LegalHoldJobSettings{
		EnableLegalHoldJobs: s.EnableLegalHoldJobs,
		TimeOfDay:           s.TimeOfDay,
	}
}

func (s *LegalHoldJobSettings) String() string {
	return fmt.Sprintf("enabled=%T, tod=%s", s.EnableLegalHoldJobs, s.TimeOfDay.Format(TimeOfDayLayout))
}

func parseLegaHoldJobSettings(cfg *config.Configuration) (*LegalHoldJobSettings, error) {
	if cfg == nil {
		return &LegalHoldJobSettings{
			EnableLegalHoldJobs: false,
		}, nil
	}

	tod, err := time.Parse(TimeOfDayLayout, cfg.TimeOfDay)
	if err != nil {
		return nil, fmt.Errorf("cannot parse `Time of day`: %w", err)
	}

	return &LegalHoldJobSettings{
		EnableLegalHoldJobs: true,
		TimeOfDay:           tod,
	}, nil
}

func (s *LegalHoldJobSettings) CalcNext(last time.Time, timeOfDay time.Time) time.Time {
	originalLocation := timeOfDay.Location()
	last = last.In(originalLocation)
	next := time.Date(last.Year(), last.Month(), last.Day(), last.Hour(), last.Minute()+2, last.Second(), 0, timeOfDay.Location())
	return next.In(originalLocation)
}
