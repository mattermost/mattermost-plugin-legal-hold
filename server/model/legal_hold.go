package model

import (
	"fmt"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"
	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/pkg/errors"
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

	return nil
}

// NeedsExecuting returns true if, at the time provided for "now", the Legal Hold is ready to
// be executed, or false if it is not yet ready to executed.
func (lh *LegalHold) NeedsExecuting(now int64) bool {
	// Calculate the execution start time.
	startTime := utils.Max(lh.LastExecutionEndedAt, lh.StartsAt)

	// Calculate the end time.
	endTime := utils.Min(startTime+lh.ExecutionLength, lh.EndsAt)

	// The legal hold is only ready to be executed if the end time is in the past relative
	// to the "now" time.
	return now > endTime
}

// IsFinished returns true if the legal hold has executed all the way to the end time or false
// if it has not.
func (lh *LegalHold) IsFinished() bool {
	return lh.LastExecutionEndedAt >= lh.EndsAt
}

type CreateLegalHold struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	UserIDs     []string `json:"user_ids"`
	StartsAt    int64    `json:"starts_at"`
	EndsAt      int64    `json:"ends_at"`
}

// NewLegalHoldFromCreate creates and populates a new LegalHold struct from
// the provided CreateLegalHold struct.
func NewLegalHoldFromCreate(lhc CreateLegalHold) LegalHold {
	return LegalHold{
		ID:                   mattermostModel.NewId(),
		Name:                 lhc.Name,
		DisplayName:          lhc.DisplayName,
		UserIDs:              lhc.UserIDs,
		StartsAt:             lhc.StartsAt,
		EndsAt:               lhc.EndsAt,
		LastExecutionEndedAt: 0,
		ExecutionLength:      864000000,
	}
}

// LegalHoldCursor represents the state of a paginated LegalHold export query.
// It is based on the model.ComplianceCursor struct from Mattermost Server.
type LegalHoldCursor struct {
	LastPostCreateAt int64
	LastPostID       string
	BatchNumber      uint
	Completed        bool
}

// NewLegalHoldCursor creates a new LegalHoldCursor object with the provided startTime
// that is initialized and ready to use.
func NewLegalHoldCursor(startTime int64) LegalHoldCursor {
	return LegalHoldCursor{
		LastPostCreateAt: startTime,
		LastPostID:       "00000000000000000000000000",
		BatchNumber:      0,
		Completed:        false,
	}
}

// LegalHoldPost represents one post and its associated data as required for a legal hold record.
// It is based on the model.CompliancePost struct from Mattermost Server.
type LegalHoldPost struct {

	// From Team
	TeamName        string `csv:"TeamName"`
	TeamDisplayName string `csv:"TeamDisplayName"`

	// From Channel
	ChannelName        string `csv:"ChannelName"`
	ChannelDisplayName string `csv:"ChannelDisplayName"`
	ChannelType        string `csv:"ChannelType"`

	// From User
	UserUsername string `csv:"UserUsername"`
	UserEmail    string `csv:"UserEmail"`
	UserNickname string `csv:"UserNickname"`

	// From Post
	PostID         string `csv:"PostId"`
	PostCreateAt   int64  `csv:"PostCreateAt"`
	PostUpdateAt   int64  `csv:"PostUpdateAt"`
	PostDeleteAt   int64  `csv:"PostDeleteAt"`
	PostRootID     string `csv:"PostRootId"`
	PostOriginalID string `csv:"PostOriginalId"`
	PostMessage    string `csv:"PostMessage"`
	PostType       string `csv:"PostType"`
	PostProps      string `csv:"PostProps"`
	PostHashtags   string `csv:"PostHashtags"`
	PostFileIDs    string `csv:"PostFileIds"`

	IsBot bool `csv:"IsBot"`
}

// LegalHoldPostHeader returns the headers for a tabulated representation of LegalHoldPost structs.
// It is based on the model.CompliancePostHeader function from Mattermost Server.
func LegalHoldPostHeader() []string {
	return []string{
		"TeamName",
		"TeamDisplayName",

		"ChannelName",
		"ChannelDisplayName",
		"ChannelType",

		"UserUsername",
		"UserEmail",
		"UserNickname",
		"UserType",

		"PostId",
		"PostCreateAt",
		"PostUpdateAt",
		"PostDeleteAt",
		"PostRootId",
		"PostOriginalId",
		"PostMessage",
		"PostType",
		"PostProps",
		"PostHashtags",
		"PostFileIds",
	}
}

type LegalHoldChannelIndex map[string][]LegalHoldChannelMembership

type LegalHoldChannelMembership struct {
	ChannelID string
	StartTime int64
	EndTime   int64
}

// Merge merges the new LegalHoldChannelIndex into this LegalHoldChannelIndex.
func (lhci *LegalHoldChannelIndex) Merge(new *LegalHoldChannelIndex) {
	for userID, newChannels := range *new {
		if oldChannels, ok := (*lhci)[userID]; !ok {
			(*lhci)[userID] = newChannels
		} else {
			var combinedChannels []LegalHoldChannelMembership
			for _, newChannel := range newChannels {
				if oldChannel, ok := getLegalHoldChannelMembership(oldChannels, newChannel.ChannelID); ok {
					// Record for channel exists in both indexes.
					combinedChannels = append(combinedChannels, oldChannel.Combine(newChannel))
				} else {
					// Record for channel only exists in new index.
					combinedChannels = append(combinedChannels, newChannel)
				}
			}

			for _, oldChannel := range oldChannels {
				if _, ok := getLegalHoldChannelMembership(newChannels, oldChannel.ChannelID); !ok {
					// Record for channel only exists in old index.
					combinedChannels = append(combinedChannels, oldChannel)
				}
			}

			(*lhci)[userID] = combinedChannels
		}
	}
}

func getLegalHoldChannelMembership(channelMemberships []LegalHoldChannelMembership, channelID string) (LegalHoldChannelMembership, bool) {
	for _, cm := range channelMemberships {
		if cm.ChannelID == channelID {
			return cm, true
		}
	}

	return LegalHoldChannelMembership{}, false
}

// Combine combines the data from two LegalHoldChannelMembership structs and returns a new one
// representing the combined result.
func (lhcm LegalHoldChannelMembership) Combine(new LegalHoldChannelMembership) LegalHoldChannelMembership {
	return LegalHoldChannelMembership{
		ChannelID: lhcm.ChannelID,
		StartTime: utils.Min(lhcm.StartTime, new.StartTime),
		EndTime:   utils.Max(lhcm.EndTime, new.EndTime),
	}
}
