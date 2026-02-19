package parse

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func TestLoadIndex(t *testing.T) {
	t.Run("loads valid index.json", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		index := model.LegalHoldIndex{
			LegalHold: model.LegalHoldIndexDetails{
				ID:          "lh1",
				Name:        "test-hold",
				DisplayName: "Test Legal Hold",
			},
			Teams: []*model.LegalHoldTeam{
				{
					ID:          "team1",
					Name:        "test-team",
					DisplayName: "Test Team",
					Channels: []*model.LegalHoldChannel{
						{
							ID:          "channel1",
							Name:        "town-square",
							DisplayName: "Town Square",
							Type:        "O",
						},
					},
				},
			},
			Users: model.LegalHoldIndexUsers{
				"user1": {
					Username: "testuser",
					Email:    "test@example.com",
					Channels: []model.LegalHoldChannelMembership{
						{
							ChannelID: "channel1",
							StartTime: 0,
							EndTime:   9999999999999,
						},
					},
				},
			},
		}

		indexJSON, err := json.Marshal(index)
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(tempDir, "index.json"), indexJSON, 0644)
		require.NoError(t, err)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		result, err := LoadIndex(legalHold)

		require.NoError(t, err)
		assert.Equal(t, "lh1", result.LegalHold.ID)
		assert.Equal(t, "Test Legal Hold", result.LegalHold.DisplayName)
		require.Len(t, result.Teams, 1)
		assert.Equal(t, "team1", result.Teams[0].ID)
		require.Len(t, result.Users, 1)
		assert.Equal(t, "testuser", result.Users["user1"].Username)
	})

	t.Run("returns error for missing index.json", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		_, err = LoadIndex(legalHold)

		require.Error(t, err)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		err = os.WriteFile(filepath.Join(tempDir, "index.json"), []byte("not valid json"), 0644)
		require.NoError(t, err)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		_, err = LoadIndex(legalHold)

		require.Error(t, err)
	})
}

func TestCreateTeamAndChannelLookup(t *testing.T) {
	t.Run("creates lookups from index with teams and channels", func(t *testing.T) {
		channel1 := &model.LegalHoldChannel{
			ID:          "channel1",
			Name:        "town-square",
			DisplayName: "Town Square",
			Type:        "O",
		}
		channel2 := &model.LegalHoldChannel{
			ID:          "channel2",
			Name:        "off-topic",
			DisplayName: "Off-Topic",
			Type:        "O",
		}
		team := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
			Channels:    []*model.LegalHoldChannel{channel1, channel2},
		}

		index := model.LegalHoldIndex{
			Teams: []*model.LegalHoldTeam{team},
		}

		teamLookup, channelLookup, teamForChannelLookup := CreateTeamAndChannelLookup(index)

		// Verify team lookup
		assert.Len(t, teamLookup, 1)
		assert.Equal(t, team, teamLookup["team1"])

		// Verify channel lookup
		assert.Len(t, channelLookup, 2)
		assert.Equal(t, channel1, channelLookup["channel1"])
		assert.Equal(t, channel2, channelLookup["channel2"])

		// Verify team for channel lookup
		assert.Len(t, teamForChannelLookup, 2)
		assert.Equal(t, team, teamForChannelLookup["channel1"])
		assert.Equal(t, team, teamForChannelLookup["channel2"])
	})

	t.Run("handles multiple teams", func(t *testing.T) {
		channel1 := &model.LegalHoldChannel{ID: "channel1", Name: "general"}
		channel2 := &model.LegalHoldChannel{ID: "channel2", Name: "random"}
		team1 := &model.LegalHoldTeam{
			ID:       "team1",
			Name:     "team-one",
			Channels: []*model.LegalHoldChannel{channel1},
		}
		team2 := &model.LegalHoldTeam{
			ID:       "team2",
			Name:     "team-two",
			Channels: []*model.LegalHoldChannel{channel2},
		}

		index := model.LegalHoldIndex{
			Teams: []*model.LegalHoldTeam{team1, team2},
		}

		teamLookup, channelLookup, teamForChannelLookup := CreateTeamAndChannelLookup(index)

		assert.Len(t, teamLookup, 2)
		assert.Len(t, channelLookup, 2)
		assert.Equal(t, team1, teamForChannelLookup["channel1"])
		assert.Equal(t, team2, teamForChannelLookup["channel2"])
	})

	t.Run("returns empty lookups for empty index", func(t *testing.T) {
		index := model.LegalHoldIndex{
			Teams: []*model.LegalHoldTeam{},
		}

		teamLookup, channelLookup, teamForChannelLookup := CreateTeamAndChannelLookup(index)

		assert.Empty(t, teamLookup)
		assert.Empty(t, channelLookup)
		assert.Empty(t, teamForChannelLookup)
	})

	t.Run("DM/GM channels are NOT in lookups", func(t *testing.T) {
		// DMs/GMs don't belong to teams, so they won't be in the index.Teams
		// This test verifies that looking up a DM channel returns nil
		channel1 := &model.LegalHoldChannel{ID: "channel1", Name: "general"}
		team := &model.LegalHoldTeam{
			ID:       "team1",
			Name:     "test-team",
			Channels: []*model.LegalHoldChannel{channel1},
		}

		index := model.LegalHoldIndex{
			Teams: []*model.LegalHoldTeam{team},
			Users: model.LegalHoldIndexUsers{
				"user1": {
					Username: "testuser",
					Channels: []model.LegalHoldChannelMembership{
						{ChannelID: "channel1"},      // Team channel - will be in lookup
						{ChannelID: "dm_channel_id"}, // DM - won't be in lookup
					},
				},
			},
		}

		_, channelLookup, teamForChannelLookup := CreateTeamAndChannelLookup(index)

		// Team channel is in lookups
		assert.NotNil(t, channelLookup["channel1"])
		assert.NotNil(t, teamForChannelLookup["channel1"])

		// DM channel is NOT in lookups - this is expected behavior
		// The caller must handle nil returns for DMs/GMs
		assert.Nil(t, channelLookup["dm_channel_id"])
		assert.Nil(t, teamForChannelLookup["dm_channel_id"])
	})
}
