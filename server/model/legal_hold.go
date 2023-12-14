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
	DeleteAt             int64    `json:"delete_at"`
	UserIDs              []string `json:"user_ids"`
	StartsAt             int64    `json:"starts_at"`
	EndsAt               int64    `json:"ends_at"`
	LastExecutionEndedAt int64    `json:"last_execution_ended_at"`
	ExecutionLength      int64    `json:"execution_length"`
}

// DeepCopy creates a deep copy of the LegalHold.
func (lh *LegalHold) DeepCopy() LegalHold {
	newLegalHold := LegalHold{
		ID:                   lh.ID,
		Name:                 lh.Name,
		DisplayName:          lh.DisplayName,
		CreateAt:             lh.CreateAt,
		UpdateAt:             lh.UpdateAt,
		DeleteAt:             lh.DeleteAt,
		StartsAt:             lh.StartsAt,
		EndsAt:               lh.EndsAt,
		LastExecutionEndedAt: lh.LastExecutionEndedAt,
		ExecutionLength:      lh.ExecutionLength,
	}

	copy(lh.UserIDs, newLegalHold.UserIDs)

	return newLegalHold
}

// IsValidForCreate checks whether the LegalHold contains data that is valid for
// creation. If it is not valid, it returns an error describing the validation
// failure. It does not guarantee that creation in the store will be successful,
// as other issues such as non-unique ID value can still cause the LegalHold to
// fail to save.
func (lh *LegalHold) IsValidForCreate() error {
	if !mattermostModel.IsValidId(lh.ID) {
		return errors.New(fmt.Sprintf("LegalHold ID is not valid: %s", lh.ID))
	}

	if !mattermostModel.IsValidAlphaNumHyphenUnderscore(lh.Name, true) {
		return errors.New(fmt.Sprintf("LegalHold Name is not valid: %s", lh.Name))
	}

	if len(lh.Name) > 64 || len(lh.Name) < 2 {
		return errors.New("LegalHold name must be between 2 and 64 characters in length")
	}

	if len(lh.DisplayName) > 64 || len(lh.DisplayName) < 2 {
		return errors.New("LegalHold display name must be between 2 and 64 characters in length")
	}

	// FIXME: More validation required here.

	return nil
}

// NeedsExecuting returns true if, at the time provided in "now", the Legal Hold is ready to
// be executed, or false if it is not yet ready to be executed.
func (lh *LegalHold) NeedsExecuting(now int64) bool {
	// Calculate the execution start time.
	startTime := utils.Max(lh.LastExecutionEndedAt, lh.StartsAt)

	// Calculate the end time.
	endTime := utils.Min(startTime+lh.ExecutionLength, lh.EndsAt)

	// The legal hold is only ready to be executed if the end time is in the past relative
	// to the "now" time.
	return now > endTime
}

// IsFinished returns true if the legal hold has executed all the way to its end time or false
// if it has not.
func (lh *LegalHold) IsFinished() bool {
	return lh.LastExecutionEndedAt >= lh.EndsAt
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

// UpdateLegalHold holds the data that is specified in teh API call to update a LegalHold.
type UpdateLegalHold struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"display_name"`
	UserIDs     []string `json:"user_ids"`
	EndsAt      int64    `json:"ends_at"`
}

func (ulh UpdateLegalHold) IsValid() error {
	if !mattermostModel.IsValidId(ulh.ID) {
		return errors.New(fmt.Sprintf("LegalHold ID is not valid: %s", ulh.ID))
	}

	if len(ulh.DisplayName) > 64 || len(ulh.DisplayName) < 2 {
		return errors.New("LegalHold display name must be between 2 and 64 characters in length")
	}

	// FIXME: More validation required here.

	return nil
}

func (lh *LegalHold) ApplyUpdates(updates UpdateLegalHold) {
	lh.DisplayName = updates.DisplayName
	lh.UserIDs = updates.UserIDs
	lh.EndsAt = updates.EndsAt
}
