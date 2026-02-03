package parse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func TestLoadPosts(t *testing.T) {
	t.Run("returns nil when messages directory does not exist", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create channel directory but no messages subdirectory
		channelDir := filepath.Join(tempDir, "channel1")
		err = os.MkdirAll(channelDir, 0755)
		require.NoError(t, err)

		channel := model.NewChannel(channelDir, "channel1")

		posts, err := LoadPosts(channel)

		require.NoError(t, err)
		assert.Nil(t, posts)
	})

	t.Run("returns empty slice when messages directory is empty", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create channel directory with empty messages subdirectory
		messagesDir := filepath.Join(tempDir, "channel1", "messages")
		err = os.MkdirAll(messagesDir, 0755)
		require.NoError(t, err)

		channel := model.NewChannel(filepath.Join(tempDir, "channel1"), "channel1")

		posts, err := LoadPosts(channel)

		require.NoError(t, err)
		assert.Empty(t, posts)
	})

	t.Run("loads posts from CSV file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		messagesDir := filepath.Join(tempDir, "channel1", "messages")
		err = os.MkdirAll(messagesDir, 0755)
		require.NoError(t, err)

		// Create a CSV file with post data
		csvContent := `TeamId,TeamName,TeamDisplayName,ChannelName,ChannelDisplayName,ChannelType,UserUsername,UserEmail,UserNickname,PostId,PostCreateAt,PostUpdateAt,PostDeleteAt,PostRootId,PostOriginalId,PostMessage,PostType,PostProps,PostHashtags,PostFileIds,IsBot
team1,test-team,Test Team,test-channel,Test Channel,O,testuser,test@example.com,Test,post1,1609459200000,1609459200000,0,,,Hello World,,{},,,false`

		err = os.WriteFile(filepath.Join(messagesDir, "posts.csv"), []byte(csvContent), 0644)
		require.NoError(t, err)

		channel := model.NewChannel(filepath.Join(tempDir, "channel1"), "channel1")

		posts, err := LoadPosts(channel)

		require.NoError(t, err)
		require.Len(t, posts, 1)
		assert.Equal(t, "post1", posts[0].PostID)
		assert.Equal(t, "Hello World", posts[0].PostMessage)
		assert.Equal(t, "testuser", posts[0].UserUsername)
	})

	t.Run("filters posts by time bounds", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		messagesDir := filepath.Join(tempDir, "channel1", "messages")
		err = os.MkdirAll(messagesDir, 0755)
		require.NoError(t, err)

		// Create CSV with posts at different times
		csvContent := `TeamId,TeamName,TeamDisplayName,ChannelName,ChannelDisplayName,ChannelType,UserUsername,UserEmail,UserNickname,PostId,PostCreateAt,PostUpdateAt,PostDeleteAt,PostRootId,PostOriginalId,PostMessage,PostType,PostProps,PostHashtags,PostFileIds,IsBot
team1,test-team,Test Team,test-channel,Test Channel,O,testuser,test@example.com,Test,post1,1000,1000,0,,,Too Early,,{},,,false
team1,test-team,Test Team,test-channel,Test Channel,O,testuser,test@example.com,Test,post2,5000,5000,0,,,In Range,,{},,,false
team1,test-team,Test Team,test-channel,Test Channel,O,testuser,test@example.com,Test,post3,9000,9000,0,,,Too Late,,{},,,false`

		err = os.WriteFile(filepath.Join(messagesDir, "posts.csv"), []byte(csvContent), 0644)
		require.NoError(t, err)

		// Create channel with time bounds that only include post2
		channel := model.NewChannelWithBounds(filepath.Join(tempDir, "channel1"), "channel1", 2000, 7000)

		posts, err := LoadPosts(channel)

		require.NoError(t, err)
		require.Len(t, posts, 1)
		assert.Equal(t, "post2", posts[0].PostID)
		assert.Equal(t, "In Range", posts[0].PostMessage)
	})

	t.Run("returns empty when all posts outside time bounds", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "legal-hold-test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		messagesDir := filepath.Join(tempDir, "channel1", "messages")
		err = os.MkdirAll(messagesDir, 0755)
		require.NoError(t, err)

		csvContent := `TeamId,TeamName,TeamDisplayName,ChannelName,ChannelDisplayName,ChannelType,UserUsername,UserEmail,UserNickname,PostId,PostCreateAt,PostUpdateAt,PostDeleteAt,PostRootId,PostOriginalId,PostMessage,PostType,PostProps,PostHashtags,PostFileIds,IsBot
team1,test-team,Test Team,test-channel,Test Channel,O,testuser,test@example.com,Test,post1,1000,1000,0,,,Message,,{},,,false`

		err = os.WriteFile(filepath.Join(messagesDir, "posts.csv"), []byte(csvContent), 0644)
		require.NoError(t, err)

		// Time bounds that exclude all posts
		channel := model.NewChannelWithBounds(filepath.Join(tempDir, "channel1"), "channel1", 5000, 9000)

		posts, err := LoadPosts(channel)

		require.NoError(t, err)
		assert.Empty(t, posts)
	})
}

func TestAddFilesToPosts(t *testing.T) {
	t.Run("returns empty slice for nil posts", func(t *testing.T) {
		fileLookup := model.FileLookup{
			"file1": "files/file1/image.png",
		}

		result := AddFilesToPosts(nil, fileLookup)

		assert.Empty(t, result)
	})

	t.Run("returns empty slice for empty posts", func(t *testing.T) {
		fileLookup := model.FileLookup{
			"file1": "files/file1/image.png",
		}

		result := AddFilesToPosts([]*model.Post{}, fileLookup)

		assert.Empty(t, result)
	})

	t.Run("handles posts with no file IDs", func(t *testing.T) {
		posts := []*model.Post{
			{
				PostID:      "post1",
				PostMessage: "No files here",
				PostFileIDs: "[]",
			},
		}
		fileLookup := model.FileLookup{}

		result := AddFilesToPosts(posts, fileLookup)

		require.Len(t, result, 1)
		assert.Equal(t, "post1", result[0].PostID)
		assert.Empty(t, result[0].Files)
	})

	t.Run("handles missing files in lookup gracefully", func(t *testing.T) {
		posts := []*model.Post{
			{
				PostID:      "post1",
				PostMessage: "Has file reference",
				PostFileIDs: "[file_not_in_lookup]",
			},
		}
		fileLookup := model.FileLookup{} // Empty lookup

		result := AddFilesToPosts(posts, fileLookup)

		require.Len(t, result, 1)
		assert.Equal(t, "post1", result[0].PostID)
		// File not in lookup should not be added
		assert.Empty(t, result[0].Files)
	})
}
