package legalhold

import (
	"bytes"
	"context"
	dbsql "database/sql"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"

	// Load the MySQL driver
	_ "github.com/go-sql-driver/mysql"
	// Load the Postgres driver
	_ "github.com/lib/pq"
)

func TestDBContainers(t *testing.T) {
	t.Run("Postgres", func(t *testing.T) {
		connStr, tearDown, err := utils.CreateTestDB(context.TODO(), "postgres", "mattermost_test")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, tearDown(context.Background()))
		}()

		t.Log("Connection string: ", connStr)

		time.Sleep(5 * time.Second)

		db, err := dbsql.Open("postgres", connStr)
		require.NoError(t, err)

		err = db.Ping()
		require.NoError(t, err)

		assert.NoError(t, db.Close())
	})

	t.Run("MySQL", func(t *testing.T) {
		connStr, tearDown, err := utils.CreateTestDB(context.TODO(), "mysql", "mattermost_test")
		require.NoError(t, err)
		defer func() {
			assert.NoError(t, tearDown(context.Background()))
		}()

		t.Log("Connection string: ", connStr)

		time.Sleep(5 * time.Second)

		db, err := dbsql.Open("mysql", connStr)
		require.NoError(t, err)

		err = db.Ping()
		require.NoError(t, err)

		assert.NoError(t, db.Close())
	})
}

func TestMinIOContainers(t *testing.T) {
	connStr, tearDown, err := utils.CreateMinio(context.TODO())
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, tearDown(context.Background()))
	}()

	fileBackendSettings := utils.GetBackendSettings(connStr)
	fileBackend, err := filestore.NewFileBackend(fileBackendSettings)
	require.NoError(t, err)
	require.NoError(t, fileBackend.TestConnection())
}

func TestApp_LegalHoldExecution_Execute(t *testing.T) {
	th := sqlstore.SetupHelper(t).SetupBasic(t)
	defer th.TearDown(t)

	const channelCount = 10
	const postCount = 10

	// create a bunch of channels
	channels, err := th.CreateChannelsWithChannelMemberHistory(channelCount, "stale-test", th.User1.Id, th.Team1.Id)
	require.NoError(t, err)

	var posts []*mattermostModel.Post

	// add some posts
	for _, channel := range channels {
		posts, err = th.CreatePostsWithAttachments(postCount, th.User1.Id, channel.Id)
		require.NoError(t, err)
	}

	_ = posts

	// create a LegalHold
	lh := model.LegalHold{
		ID:                   mattermostModel.NewId(),
		UserIDs:              []string{th.User1.Id},
		StartsAt:             mattermostModel.GetMillis() - 10000,
		EndsAt:               mattermostModel.GetMillis() + 10000,
		LastExecutionEndedAt: 0,
		ExecutionLength:      1000000,
	}

	// FIXME: Update and reinstate these tests.
	_ = lh
	// lhe := NewExecution(lh, th.Store, th.FileBackend)
	// err = lhe.Execute()
	// require.Greater(t, len(lhe.channelIDs), 1)
	// require.NoError(t, err)

	// TODO: Do some proper assertions here to really test the functionality.
}

func TestExecution_FilePath(t *testing.T) {
	ex := &Execution{
		LegalHold: &model.LegalHold{
			ID:   "abc123",
			Name: "holdname",
		},
	}
	const (
		channelID     = "channelid1"
		batchCreateAt = int64(1700000000000)
		batchPostID   = "postid1"
		fileID        = "fileid1"
	)
	basePath := ex.LegalHold.BasePath()
	expectedPrefix := basePath + "/"

	testCases := []struct {
		name      string
		fileName  string
		expectErr bool
		expected  string
	}{
		{
			name:     "normal filename",
			fileName: "document.pdf",
			expected: basePath + "/channelid1/files/files-1700000000000-postid1/fileid1/document.pdf",
		},
		{
			name:     "filename with spaces",
			fileName: "my document.pdf",
			expected: basePath + "/channelid1/files/files-1700000000000-postid1/fileid1/my document.pdf",
		},
		{
			name:      "path traversal with forward slashes",
			fileName:  "../../../../plugins/com.mattermost.plugin-legal-hold.tar.gz",
			expectErr: false,
			expected:  basePath + "/channelid1/files/files-1700000000000-postid1/fileid1/com.mattermost.plugin-legal-hold.tar.gz",
		},
		{
			name:      "absolute path",
			fileName:  "/etc/passwd",
			expectErr: false,
			expected:  basePath + "/channelid1/files/files-1700000000000-postid1/fileid1/passwd",
		},
		{
			name:      "single dot-dot",
			fileName:  "..",
			expectErr: true,
		},
		{
			name:      "single dot",
			fileName:  ".",
			expectErr: true,
		},
		{
			name:      "empty name",
			fileName:  "",
			expectErr: true,
		},
		{
			name:      "just a slash",
			fileName:  "/",
			expectErr: true,
		},
		{
			name:      "trailing slash gets cleaned",
			fileName:  "foo/",
			expectErr: false,
			expected:  basePath + "/channelid1/files/files-1700000000000-postid1/fileid1/foo",
		},
		{
			name:      "filename with embedded traversal via backslashes (linux-safe)",
			fileName:  "..\\..\\plugins\\evil.tar.gz",
			expectErr: false,
			// On Linux, backslashes are valid filename characters; Base returns the whole string.
			expected: basePath + "/channelid1/files/files-1700000000000-postid1/fileid1/..\\..\\plugins\\evil.tar.gz",
		},
		{
			name:      "ends with dot-dot segment",
			fileName:  "foo/..",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ex.filePath(channelID, batchCreateAt, batchPostID, fileID, tc.fileName)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, got)
			require.True(t, strings.HasPrefix(got, expectedPrefix),
				"path %q must stay inside base %q", got, basePath)
		})
	}
}

func TestLegalHold_Hash(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		secret         string
		expectedOutput string
		expectedError  error
	}{
		{
			name:           "empty input",
			input:          "",
			secret:         "foo",
			expectedOutput: "1c96626e0b37265976336cc10499fe1f48f9e591d68a48814c8c937613795929634c615a3fee488fd77526a352d1809f7a4e5bc1075bdc8f52e7d6cae0775a46",
			expectedError:  nil,
		},
		{
			name:           "valid input",
			input:          "Hello, World!",
			secret:         "foo",
			expectedOutput: "408dce73e276f62b584901c4b6395ce19c49e0c54982bb3f972cbb28579086138741362f07302817db0b158a03ffbd63a99169a87a897343be63e201269d11ef",
			expectedError:  nil,
		},
		{
			name:           "valid input",
			input:          "2",
			secret:         "",
			expectedOutput: "0b68cd17f7c117926dcfc0f993d723c09b2f8bd5b4689a0dc6ff4577db06ebd7fa3b70290a6b8e3ea89550b12b6f2e50403290d8e1e70bac4316396359b51cdd",
			expectedError:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(tc.input))
			result, err := hashFromReader(tc.secret, reader)

			if err != nil {
				if tc.expectedError == nil {
					t.Errorf("hash() with args %v : Unexpected error %v", tc.input, err)
				} else if err.Error() != tc.expectedError.Error() {
					t.Errorf("hash() with args %v : expected %v, got %v",
						tc.input, tc.expectedError, err)
				}
			} else {
				if tc.expectedOutput != result {
					t.Errorf("hash() with args %v : expected %v, got %v",
						tc.input, tc.expectedOutput, result)
				}
			}
		})
	}
}
