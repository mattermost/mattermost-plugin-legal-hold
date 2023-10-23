package legalhold

import (
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

	lhe := NewExecution(lh, th.Store, th.FileBackend)
	err = lhe.Execute()
	require.Greater(t, len(lhe.channelIDs), 1)
	require.NoError(t, err)

	// TODO: Do some proper assertions here to really test the functionality.
}
