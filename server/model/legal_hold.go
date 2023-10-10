package model

import "github.com/mattermost/mattermost-plugin-legal-hold/server/utils"

// LegalHold represents one legal hold.
type LegalHold struct {
	ID                   string
	UserIDs              []string
	StartsAt             int64
	EndsAt               int64
	LastExecutionEndedAt int64
	ExecutionLength      int64
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
// that is initialised and ready to use.
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
	PostId         string `csv:"PostId"`
	PostCreateAt   int64  `csv:"PostCreateAt"`
	PostUpdateAt   int64  `csv:"PostUpdateAt"`
	PostDeleteAt   int64  `csv:"PostDeleteAt"`
	PostRootId     string `csv:"PostRootId"`
	PostOriginalId string `csv:"PostOriginalId"`
	PostMessage    string `csv:"PostMessage"`
	PostType       string `csv:"PostType"`
	PostProps      string `csv:"PostProps"`
	PostHashtags   string `csv:"PostHashtags"`
	PostFileIds    string `csv:"PostFileIds"`

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
