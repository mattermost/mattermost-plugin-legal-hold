package model

import (
	"fmt"

	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"
)

// LegalHold represents one legal hold.
type LegalHold struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	DisplayName          string   `json:"display_name"`
	CreateAt             int64    `json:"create_at"`
	UpdateAt             int64    `json:"update_at"`
	UserIDs              []string `json:"user_ids"`
	StartsAt             int64    `json:"starts_at"`
	EndsAt               int64    `json:"ends_at"`
	LastExecutionEndedAt int64    `json:"last_execution_ended_at"`
	ExecutionLength      int64    `json:"execution_length"`
}

// DeepCopy creates a deep copy of the LegalHold.
func (lh *LegalHold) DeepCopy() LegalHold {
	if lh == nil {
		return LegalHold{}
	}

	newLegalHold := LegalHold{
		ID:                   lh.ID,
		Name:                 lh.Name,
		DisplayName:          lh.DisplayName,
		CreateAt:             lh.CreateAt,
		UpdateAt:             lh.UpdateAt,
		StartsAt:             lh.StartsAt,
		EndsAt:               lh.EndsAt,
		LastExecutionEndedAt: lh.LastExecutionEndedAt,
		ExecutionLength:      lh.ExecutionLength,
	}

	if len(lh.UserIDs) > 0 {
		newLegalHold.UserIDs = make([]string, len(lh.UserIDs))
		copy(newLegalHold.UserIDs, lh.UserIDs)
	}

	return newLegalHold
}

// IsValidForCreate checks whether the LegalHold contains data that is valid for
// creation. If it is not valid, it returns an error describing the validation
// failure. It does not guarantee that creation in the store will be successful,
// as other issues such as non-unique ID value can still cause the LegalHold to
// fail to save.
func (lh *LegalHold) IsValidForCreate() error {
	if !mattermostModel.IsValidId(lh.ID) {
		return fmt.Errorf("LegalHold ID is not valid: %s", lh.ID)
	}

	if !mattermostModel.IsValidAlphaNumHyphenUnderscore(lh.Name, true) {
		return fmt.Errorf("LegalHold Name is not valid: %s", lh.Name)
	}

	if len(lh.Name) > 64 || len(lh.Name) < 2 {
		return errors.New("LegalHold name must be between 2 and 64 characters in length")
	}

	if len(lh.DisplayName) > 64 || len(lh.DisplayName) < 2 {
		return errors.New("LegalHold display name must be between 2 and 64 characters in length")
	}

	if lh.UserIDs == nil || len(lh.UserIDs) < 1 {
		return errors.New("LegalHold must include at least 1 user")
	}

	for _, userID := range lh.UserIDs {
		if !mattermostModel.IsValidId(userID) {
			return errors.New("LegalHold users must have valid IDs")
		}
	}

	if lh.StartsAt < 1 {
		return errors.New("LegalHold must start at a valid time")
	}

	if lh.EndsAt < 0 {
		return errors.New("LegalHold must end at a valid time or zero")
	}

	return nil
}

// NeedsExecuting returns true if, at the time provided in "now", the Legal Hold is ready to
// be executed, or false if it is not yet ready to be executed.
func (lh *LegalHold) NeedsExecuting(now int64) bool {
	// The legal hold is only ready to be executed if the NextExecutionEndTime is
	// in the past relative to the time "now".
	return now > lh.NextExecutionEndTime()
}

// NextExecutionStartTime returns the time at which the next execution of this
// LegalHold should start.
func (lh *LegalHold) NextExecutionStartTime() int64 {
	return utils.Max(lh.LastExecutionEndedAt, lh.StartsAt)
}

// NextExecutionEndTime returns th etime at which the next execution of this
// LegalHold should end.
func (lh *LegalHold) NextExecutionEndTime() int64 {
	endTime := lh.NextExecutionStartTime() + lh.ExecutionLength
	if lh.EndsAt > 0 {
		endTime = utils.Min(endTime, lh.EndsAt)
	}
	return endTime
}

// IsFinished returns true if the legal hold has executed all the way to its end time or false
// if it has not.
func (lh *LegalHold) IsFinished() bool {
	return lh.EndsAt != 0 && lh.LastExecutionEndedAt >= lh.EndsAt
}

// BasePath returns the base file storage path for this legal hold.
func (lh *LegalHold) BasePath() string {
	return fmt.Sprintf("legal_hold/%s_(%s)", lh.Name, lh.ID)
}

// CreateLegalHold holds the data that is specified in the API call to create a LegalHold.
type CreateLegalHold struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	UserIDs     []string `json:"user_ids"`
	StartsAt    int64    `json:"starts_at"`
	EndsAt      int64    `json:"ends_at"`
}

// NewLegalHoldFromCreate creates and populates a new LegalHold instance from
// the provided CreateLegalHold instance.
func NewLegalHoldFromCreate(lhc CreateLegalHold) LegalHold {
	return LegalHold{
		ID:                   mattermostModel.NewId(),
		Name:                 lhc.Name,
		DisplayName:          lhc.DisplayName,
		UserIDs:              lhc.UserIDs,
		StartsAt:             lhc.StartsAt,
		EndsAt:               lhc.EndsAt,
		LastExecutionEndedAt: 0,
		ExecutionLength:      86400000,
	}
}

// UpdateLegalHold holds the data that is specified in the API call to update a LegalHold.
type UpdateLegalHold struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	UserIDs     []string `json:"user_ids"`
	EndsAt      int64    `json:"ends_at"`
}

func (ulh UpdateLegalHold) IsValid() error {
	if !mattermostModel.IsValidId(ulh.ID) {
		return fmt.Errorf("LegalHold ID is not valid: %s", ulh.ID)
	}

	if len(ulh.DisplayName) > 64 || len(ulh.DisplayName) < 2 {
		return errors.New("LegalHold display name must be between 2 and 64 characters in length")
	}

	if ulh.UserIDs == nil || len(ulh.UserIDs) < 1 {
		return errors.New("LegalHold must include at least 1 user")
	}

	for _, userID := range ulh.UserIDs {
		if !mattermostModel.IsValidId(userID) {
			return errors.New("LegalHold users must have valid IDs")
		}
	}

	if ulh.EndsAt < 0 {
		return errors.New("LegalHold must end at a valid time or zero")
	}

	return nil
}

func (lh *LegalHold) ApplyUpdates(updates UpdateLegalHold) {
	lh.DisplayName = updates.DisplayName
	lh.UserIDs = updates.UserIDs
	lh.EndsAt = updates.EndsAt
}
