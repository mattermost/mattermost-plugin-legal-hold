package view

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func TestWriteIndexFile(t *testing.T) {
	t.Run("handles DM/GM channels not in team lookup without crashing", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		// Create index with a user who has a DM channel membership
		legalHoldIndex := model.LegalHoldIndex{
			LegalHold: model.LegalHoldIndexDetails{
				ID:          "lh1",
				Name:        "test-hold",
				DisplayName: "Test Legal Hold",
			},
			Teams: []*model.LegalHoldTeam{},
			Users: model.LegalHoldIndexUsers{
				"user1": {
					Username: "testuser",
					Email:    "test@example.com",
					Channels: []model.LegalHoldChannelMembership{
						{
							ChannelID: "dm_channel_not_in_team",
							StartTime: 0,
							EndTime:   9999999999999,
						},
					},
				},
			},
		}

		// Empty lookups - simulating DM/GM not being in any team
		teamLookup := model.TeamLookup{}
		channelLookup := model.ChannelLookup{}
		teamForChannelLookup := model.TeamForChannelLookup{}

		// This should not panic
		err = WriteIndexFile(legalHold, legalHoldIndex, teamLookup, channelLookup, teamForChannelLookup, tempDir)
		require.NoError(t, err)

		// Verify the index.html was created
		indexPath := filepath.Join(tempDir, "index.html")
		_, err = os.Stat(indexPath)
		require.NoError(t, err)

		// Read and verify content includes fallback values
		content, err := os.ReadFile(indexPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "testuser")
		assert.Contains(t, string(content), "Direct Messages")
	})

	t.Run("handles mix of team channels and DM/GM channels", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		teamChannel := &model.LegalHoldChannel{
			ID:          "team_channel_1",
			Name:        "town-square",
			DisplayName: "Town Square",
			Type:        "O",
		}

		team := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
			Channels:    []*model.LegalHoldChannel{teamChannel},
		}

		legalHoldIndex := model.LegalHoldIndex{
			LegalHold: model.LegalHoldIndexDetails{
				ID:          "lh1",
				Name:        "test-hold",
				DisplayName: "Test Legal Hold",
			},
			Teams: []*model.LegalHoldTeam{team},
			Users: model.LegalHoldIndexUsers{
				"user1": {
					Username: "testuser",
					Email:    "test@example.com",
					Channels: []model.LegalHoldChannelMembership{
						{
							ChannelID: "team_channel_1",
							StartTime: 0,
							EndTime:   9999999999999,
						},
						{
							ChannelID: "dm_channel_id",
							StartTime: 0,
							EndTime:   9999999999999,
						},
					},
				},
			},
		}

		teamLookup := model.TeamLookup{
			"team1": team,
		}
		channelLookup := model.ChannelLookup{
			"team_channel_1": teamChannel,
		}
		teamForChannelLookup := model.TeamForChannelLookup{
			"team_channel_1": team,
		}

		// This should not panic even though dm_channel_id is not in lookups
		err = WriteIndexFile(legalHold, legalHoldIndex, teamLookup, channelLookup, teamForChannelLookup, tempDir)
		require.NoError(t, err)

		// Verify the index.html was created
		indexPath := filepath.Join(tempDir, "index.html")
		content, err := os.ReadFile(indexPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Test Team")
		assert.Contains(t, contentStr, "Town Square")
		assert.Contains(t, contentStr, "Direct Messages")
		assert.Contains(t, contentStr, "testuser")
	})

	t.Run("generates correct links for user channel pages", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		legalHoldIndex := model.LegalHoldIndex{
			LegalHold: model.LegalHoldIndexDetails{
				ID:          "lh1",
				Name:        "test-hold",
				DisplayName: "Test Legal Hold",
			},
			Teams: []*model.LegalHoldTeam{},
			Users: model.LegalHoldIndexUsers{
				"user123": {
					Username: "testuser",
					Email:    "test@example.com",
					Channels: []model.LegalHoldChannelMembership{
						{
							ChannelID: "channel456",
							StartTime: 0,
							EndTime:   9999999999999,
						},
					},
				},
			},
		}

		err = WriteIndexFile(legalHold, legalHoldIndex, model.TeamLookup{}, model.ChannelLookup{}, model.TeamForChannelLookup{}, tempDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tempDir, "index.html"))
		require.NoError(t, err)

		contentStr := string(content)
		// Verify user "All Messages" link
		assert.Contains(t, contentStr, "user123.html")
		// Verify user-channel specific link
		assert.Contains(t, contentStr, "user123_channel456.html")
	})
}

func TestWriteChannel(t *testing.T) {
	t.Run("creates HTML file for channel with no posts", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		channel := model.Channel{
			ID:   "channel1",
			Path: filepath.Join(tempDir, "channel1"),
		}

		teamData := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
		}

		channelData := &model.LegalHoldChannel{
			ID:          "channel1",
			Name:        "test-channel",
			DisplayName: "Test Channel",
			Type:        "O",
		}

		// Empty posts - simulating channel with no messages
		var posts []*model.PostWithFiles

		err = WriteChannel(legalHold, channel, posts, teamData, channelData, tempDir)
		require.NoError(t, err)

		// Verify the channel HTML was created
		channelPath := filepath.Join(tempDir, "channel1.html")
		content, err := os.ReadFile(channelPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Test Channel")
		assert.Contains(t, contentStr, "Test Team")
		assert.Contains(t, contentStr, "No messages were recorded in this channel during the legal hold period")
	})

	t.Run("creates HTML file for channel with posts", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		channel := model.Channel{
			ID:   "channel1",
			Path: filepath.Join(tempDir, "channel1"),
		}

		teamData := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
		}

		channelData := &model.LegalHoldChannel{
			ID:          "channel1",
			Name:        "test-channel",
			DisplayName: "Test Channel",
			Type:        "O",
		}

		posts := []*model.PostWithFiles{
			{
				Post: &model.Post{
					PostID:       "post1",
					PostMessage:  "Hello, world!",
					PostCreateAt: 1609459200000,
					UserUsername: "testuser",
				},
				Files: []string{},
			},
		}

		err = WriteChannel(legalHold, channel, posts, teamData, channelData, tempDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tempDir, "channel1.html"))
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Hello, world!")
		assert.Contains(t, contentStr, "@testuser")
		assert.False(t, strings.Contains(contentStr, "No messages were recorded"))
	})
}

func TestWriteUserAllChannels(t *testing.T) {
	t.Run("creates HTML file for user with no posts in any channel", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		user := model.User{
			ID:       "user1",
			Username: "testuser",
			Email:    "test@example.com",
		}

		// Empty posts for all channels
		allPosts := map[string][]*model.PostWithFiles{
			"channel1": {},
			"channel2": {},
		}

		teamData := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
		}
		channelData1 := &model.LegalHoldChannel{
			ID:          "channel1",
			Name:        "general",
			DisplayName: "General",
			Type:        "O",
		}
		channelData2 := &model.LegalHoldChannel{
			ID:          "channel2",
			Name:        "random",
			DisplayName: "Random",
			Type:        "O",
		}

		teamForChannelLookup := model.TeamForChannelLookup{
			"channel1": teamData,
			"channel2": teamData,
		}
		channelLookup := model.ChannelLookup{
			"channel1": channelData1,
			"channel2": channelData2,
		}

		err = WriteUserAllChannels(legalHold, user, allPosts, teamForChannelLookup, channelLookup, tempDir)
		require.NoError(t, err)

		// Verify the user HTML was created
		userPath := filepath.Join(tempDir, "user1.html")
		content, err := os.ReadFile(userPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "testuser")
		assert.Contains(t, contentStr, "No messages were recorded")
	})

	t.Run("handles DM/GM channels not in lookups", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		user := model.User{
			ID:       "user1",
			Username: "testuser",
			Email:    "test@example.com",
		}

		// Posts for a DM channel that's not in the lookups
		allPosts := map[string][]*model.PostWithFiles{
			"dm_channel_id": {
				{
					Post: &model.Post{
						PostID:       "post1",
						PostMessage:  "Hello in DM",
						PostCreateAt: 1609459200000,
						UserUsername: "testuser",
					},
					Files: []string{},
				},
			},
		}

		// Empty lookups - DM channel won't be found
		teamForChannelLookup := model.TeamForChannelLookup{}
		channelLookup := model.ChannelLookup{}

		// This should not panic - should use fallback data
		err = WriteUserAllChannels(legalHold, user, allPosts, teamForChannelLookup, channelLookup, tempDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tempDir, "user1.html"))
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Hello in DM")
		assert.Contains(t, contentStr, "testuser")
	})

	t.Run("handles mix of team channels and DM/GM channels", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		user := model.User{
			ID:       "user1",
			Username: "testuser",
			Email:    "test@example.com",
		}

		allPosts := map[string][]*model.PostWithFiles{
			"team_channel": {
				{
					Post: &model.Post{
						PostID:       "post1",
						PostMessage:  "Team message",
						PostCreateAt: 1609459200000,
						UserUsername: "testuser",
					},
					Files: []string{},
				},
			},
			"dm_channel": {
				{
					Post: &model.Post{
						PostID:       "post2",
						PostMessage:  "DM message",
						PostCreateAt: 1609459200000,
						UserUsername: "otheruser",
					},
					Files: []string{},
				},
			},
		}

		teamData := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
		}
		channelData := &model.LegalHoldChannel{
			ID:          "team_channel",
			Name:        "general",
			DisplayName: "General",
			Type:        "O",
		}

		// Only team channel is in lookups
		teamForChannelLookup := model.TeamForChannelLookup{
			"team_channel": teamData,
		}
		channelLookup := model.ChannelLookup{
			"team_channel": channelData,
		}

		err = WriteUserAllChannels(legalHold, user, allPosts, teamForChannelLookup, channelLookup, tempDir)
		require.NoError(t, err)

		content, err := os.ReadFile(filepath.Join(tempDir, "user1.html"))
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Team message")
		assert.Contains(t, contentStr, "DM message")
	})
}

func TestWriteUserChannel(t *testing.T) {
	t.Run("creates HTML file for user channel with no posts", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		legalHold := model.LegalHold{
			ID:   "lh1",
			Name: "Test Legal Hold",
			Path: tempDir,
		}

		user := model.User{
			ID:       "user1",
			Username: "testuser",
			Email:    "test@example.com",
		}

		channel := model.Channel{
			ID:   "channel1",
			Path: filepath.Join(tempDir, "channel1"),
		}

		teamData := &model.LegalHoldTeam{
			ID:          "team1",
			Name:        "test-team",
			DisplayName: "Test Team",
		}

		channelData := &model.LegalHoldChannel{
			ID:          "channel1",
			Name:        "test-channel",
			DisplayName: "Test Channel",
			Type:        "O",
		}

		var posts []*model.PostWithFiles

		err = WriteUserChannel(legalHold, user, channel, posts, teamData, channelData, tempDir)
		require.NoError(t, err)

		// Verify the user channel HTML was created with correct naming
		userChannelPath := filepath.Join(tempDir, "user1_channel1.html")
		content, err := os.ReadFile(userChannelPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "Test Channel")
		assert.Contains(t, contentStr, "@testuser")
		assert.Contains(t, contentStr, "No messages were recorded in this channel during the user's membership")
	})
}
