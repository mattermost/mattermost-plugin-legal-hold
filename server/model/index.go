package model

import "github.com/mattermost/mattermost-plugin-legal-hold/server/utils"

// LegalHoldChannelMembership represents the membership of a channel by a user in the
// LegalHoldIndexUsers.
type LegalHoldChannelMembership struct {
	ChannelID string `json:"channel_id"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
}

// LegalHoldIndexUser represents the data about one user in the LegalHoldIndexUsers.
type LegalHoldIndexUser struct {
	Username string                       `json:"username"`
	Email    string                       `json:"email"`
	Channels []LegalHoldChannelMembership `json:"channels"`
}

// LegalHoldIndexUsers maps to the contents of the index.json file in a legal hold export.
// It contains various pieces of metadata to help with the programmatic and manual processing of
// the legal hold export.
type LegalHoldIndexUsers map[string]LegalHoldIndexUser

type LegalHoldIndexDetails struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	DisplayName          string `json:"display_name"`
	StartsAt             int64  `json:"starts_at"`
	LastExecutionEndedAt int64  `json:"last_execution_ended_at"`
}

type LegalHoldIndex struct {
	Users     LegalHoldIndexUsers   `json:"users"`
	LegalHold LegalHoldIndexDetails `json:"legal_hold"`
	Teams     []*LegalHoldTeam      `json:"teams"`
}

type LegalHoldTeam struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	DisplayName string              `json:"display_name"`
	Channels    []*LegalHoldChannel `json:"channels"`
}

type LegalHoldChannel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Type        string `json:"type"`
}

func NewLegalHoldIndex() LegalHoldIndex {
	return LegalHoldIndex{
		Users:     make(LegalHoldIndexUsers),
		LegalHold: LegalHoldIndexDetails{},
		Teams:     make([]*LegalHoldTeam, 0),
	}
}

// Merge merges the new LegalHoldIndex into this LegalHoldIndex.
func (lhi *LegalHoldIndex) Merge(newHold *LegalHoldIndex) {
	// To merge the LegalHold data we overwrite the old struct in full
	// with the new one.
	lhi.LegalHold = newHold.LegalHold

	// Recursively merge the Teams (and their Channels) property, taking
	// the newest version for the union of both lists.
	for _, newTeam := range newHold.Teams {
		found := false
		for _, oldTeam := range lhi.Teams {
			if newTeam.ID == oldTeam.ID {
				oldTeam.Merge(newTeam)
				found = true
				break
			}
		}

		if !found {
			lhi.Teams = append(lhi.Teams, newTeam)
		}
	}

	lhi.Users.Merge(&newHold.Users)
}

// Merge merges the new LegalHoldTeam into this LegalHoldTeam.
func (team *LegalHoldTeam) Merge(newHold *LegalHoldTeam) {
	team.Name = newHold.Name
	team.DisplayName = newHold.DisplayName

	for _, newChannel := range newHold.Channels {
		found := false
		for _, oldChannel := range team.Channels {
			if newChannel.ID == oldChannel.ID {
				oldChannel.Name = newChannel.Name
				oldChannel.DisplayName = newChannel.DisplayName
				found = true
				break
			}
		}

		if !found {
			team.Channels = append(team.Channels, newChannel)
		}
	}
}

// Merge merges the new LegalHoldIndexUsers into this LegalHoldIndexUsers.
func (lhi *LegalHoldIndexUsers) Merge(newHold *LegalHoldIndexUsers) {
	for userID, newUser := range *newHold {
		if oldUser, ok := (*lhi)[userID]; !ok {
			(*lhi)[userID] = newUser
		} else {
			var combinedChannels []LegalHoldChannelMembership
			for _, newChannel := range newUser.Channels {
				if oldChannel, ok := getLegalHoldChannelMembership(oldUser.Channels, newChannel.ChannelID); ok {
					// Record for channel exists in both indexes.
					combinedChannels = append(combinedChannels, oldChannel.Combine(newChannel))
				} else {
					// Record for channel only exists in new index.
					combinedChannels = append(combinedChannels, newChannel)
				}
			}

			for _, oldChannel := range oldUser.Channels {
				if _, ok := getLegalHoldChannelMembership(newUser.Channels, oldChannel.ChannelID); !ok {
					// Record for channel only exists in old index.
					combinedChannels = append(combinedChannels, oldChannel)
				}
			}

			(*lhi)[userID] = LegalHoldIndexUser{
				Username: newUser.Username,
				Email:    newUser.Email,
				Channels: combinedChannels,
			}
		}
	}
}

// getLegalHoldChannelMembership finds and returns the LegalHoldChannelMembership from
// channelMemberships and true for the channel indicated by channelID, or returns an empty
// LegalHoldChannelMembership and false if no LegalHoldChannelMembership is found.
func getLegalHoldChannelMembership(channelMemberships []LegalHoldChannelMembership, channelID string) (LegalHoldChannelMembership, bool) {
	for _, cm := range channelMemberships {
		if cm.ChannelID == channelID {
			return cm, true
		}
	}

	return LegalHoldChannelMembership{}, false
}

// Combine combines the data from two LegalHoldChannelMembership instances and returns a new one
// representing the combined data.
func (lhcm LegalHoldChannelMembership) Combine(new LegalHoldChannelMembership) LegalHoldChannelMembership {
	return LegalHoldChannelMembership{
		ChannelID: lhcm.ChannelID,
		StartTime: utils.Min(lhcm.StartTime, new.StartTime),
		EndTime:   utils.Max(lhcm.EndTime, new.EndTime),
	}
}
