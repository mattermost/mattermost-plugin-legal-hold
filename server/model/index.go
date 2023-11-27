package model

import "github.com/mattermost/mattermost-plugin-legal-hold/server/utils"

// LegalHoldChannelMembership represents the membership of a channel by a user in the
// LegalHoldIndex.
type LegalHoldChannelMembership struct {
	ChannelID string
	StartTime int64
	EndTime   int64
}

// LegalHoldIndexUser represents the data about one user in the LegalHoldIndex.
type LegalHoldIndexUser struct {
	Username string
	Email    string
	Channels []LegalHoldChannelMembership
}

// LegalHoldIndex maps to the contents of the index.json file in a legal hold export.
// It contains various pieces of metadata to help with the programmatic and manual processing of
// the legal hold export.
type LegalHoldIndex map[string]LegalHoldIndexUser

// Merge merges the new LegalHoldIndex into this LegalHoldIndex.
func (lhi *LegalHoldIndex) Merge(new *LegalHoldIndex) {
	for userID, newUser := range *new {
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
