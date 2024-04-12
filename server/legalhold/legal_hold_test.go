package legalhold

import (
	"bytes"
	"testing"

	mattermostModel "github.com/mattermost/mattermost-server/v6/model"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

func TestApp_LegalHoldExecution_Execute(t *testing.T) {
	th := SetupHelper(t).SetupBasic(t)
	defer th.TearDown()

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
			expectedOutput: "f7fbba6e0636f890e56fbbf3283e524c6fa3204ae298382d624741d0dc6638326e282c41be5e4254d8820772c5518a2c5a8c0c7f7eda19594a7eb539453e1ed7",
			expectedError:  nil,
		},
		{
			name:           "valid input",
			input:          "Hello, World!",
			secret:         "foo",
			expectedOutput: "1964d4b69d3631a6ff90143c75ae1fb5c5c6045600c0dc52f1db1e2155028b56159d5d281479221f4d38fee22239dab46528424c2b122b62c97e75f01f409f4d",
			expectedError:  nil,
		},
		{
			name:           "valid input",
			input:          "Hello, World!",
			secret:         "",
			expectedOutput: "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387",
			expectedError:  nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader([]byte(tc.input))
			result, err := hash(tc.secret, reader)

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
