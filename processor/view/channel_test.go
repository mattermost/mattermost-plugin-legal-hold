package view

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func TestGetChannelAndTeamData(t *testing.T) {
	t.Run("channel and team found in lookups", func(t *testing.T) {
		channelLookup := model.ChannelLookup{
			"channel1": {
				ID:          "channel1",
				Name:        "test-channel",
				DisplayName: "Test Channel",
				Type:        "O",
			},
		}
		teamForChannelLookup := model.TeamForChannelLookup{
			"channel1": {
				ID:          "team1",
				Name:        "test-team",
				DisplayName: "Test Team",
			},
		}

		channelData, teamData := GetChannelAndTeamData("channel1", nil, channelLookup, teamForChannelLookup)

		require.NotNil(t, channelData)
		require.NotNil(t, teamData)
		assert.Equal(t, "channel1", channelData.ID)
		assert.Equal(t, "Test Channel", channelData.DisplayName)
		assert.Equal(t, "team1", teamData.ID)
		assert.Equal(t, "Test Team", teamData.DisplayName)
	})

	t.Run("DM/GM channel not in lookups with firstPost", func(t *testing.T) {
		channelLookup := model.ChannelLookup{}
		teamForChannelLookup := model.TeamForChannelLookup{}
		firstPost := &model.Post{
			TeamID:             "",
			TeamName:           "",
			TeamDisplayName:    "",
			ChannelName:        "user1__user2",
			ChannelDisplayName: "user1, user2",
			ChannelType:        "D",
		}

		channelData, teamData := GetChannelAndTeamData("dm_channel_id", firstPost, channelLookup, teamForChannelLookup)

		require.NotNil(t, channelData)
		require.NotNil(t, teamData)
		assert.Equal(t, "dm_channel_id", channelData.ID)
		assert.Equal(t, "user1, user2", channelData.DisplayName)
		assert.Equal(t, "D", channelData.Type)
		// When firstPost is provided, team data comes from the post (empty for DMs/GMs)
		assert.Equal(t, "", teamData.ID)
		assert.Equal(t, "", teamData.DisplayName)
	})

	t.Run("DM/GM channel not in lookups without firstPost", func(t *testing.T) {
		channelLookup := model.ChannelLookup{}
		teamForChannelLookup := model.TeamForChannelLookup{}

		channelData, teamData := GetChannelAndTeamData("dm_channel_id", nil, channelLookup, teamForChannelLookup)

		require.NotNil(t, channelData)
		require.NotNil(t, teamData)
		assert.Equal(t, "dm_channel_id", channelData.ID)
		assert.Equal(t, "Direct/Group Message", channelData.DisplayName)
		assert.Equal(t, "D", channelData.Type)
		assert.Equal(t, "", teamData.ID)
		assert.Equal(t, "No Team", teamData.DisplayName)
	})

	t.Run("channel in lookup but team not in lookup", func(t *testing.T) {
		channelLookup := model.ChannelLookup{
			"channel1": {
				ID:          "channel1",
				Name:        "test-channel",
				DisplayName: "Test Channel",
				Type:        "O",
			},
		}
		teamForChannelLookup := model.TeamForChannelLookup{}

		channelData, teamData := GetChannelAndTeamData("channel1", nil, channelLookup, teamForChannelLookup)

		require.NotNil(t, channelData)
		require.NotNil(t, teamData)
		assert.Equal(t, "channel1", channelData.ID)
		assert.Equal(t, "Test Channel", channelData.DisplayName)
		assert.Equal(t, "", teamData.ID)
		assert.Equal(t, "No Team", teamData.DisplayName)
	})
}
