package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/require"

	mattermostModel "github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

func TestSQLStore_GetPostsBatch(t *testing.T) {
	th := SetupHelper(t).SetupBasic(t)
	defer th.TearDown(t)

	const postCount = 10

	// Test with an open channel first

	// create an open channel
	channel, err := th.CreateOpenChannel("stale-test", th.User1.Id, th.Team1.Id)
	require.NoError(t, err)

	var posts []*mattermostModel.Post

	// add some posts
	posts, err = th.CreatePosts(postCount, th.User1.Id, channel.Id)
	require.NoError(t, err)

	_ = posts

	cursor := model.NewLegalHoldCursor(mattermostModel.GetMillis() - 1000000)

	var legalHold []model.LegalHoldPost
	legalHold, _, err = th.Store.GetPostsBatch(channel.Id, mattermostModel.GetMillis(), cursor, 1000)
	require.NoError(t, err)
	for _, legalHoldItem := range legalHold {
		t.Log(legalHoldItem)
	}

	// TODO: Assert on result contents

	// Test with a DM channel

	// create a DM channel and add some posts
	var directChannel *mattermostModel.Channel
	directChannel, err = th.CreateDirectMessageChannel(th.User1, th.User2)
	require.NoError(t, err)

	// populate it with some posts
	_, err = th.CreatePosts(postCount, th.User1.Id, directChannel.Id)
	require.NoError(t, err)

	cursor = model.NewLegalHoldCursor(mattermostModel.GetMillis() - 1000000)

	legalHold, _, err = th.Store.GetPostsBatch(directChannel.Id, mattermostModel.GetMillis(), cursor, 1000)
	require.NoError(t, err)
	for _, legalHoldItem := range legalHold {
		t.Log(legalHoldItem)
	}

	// TODO: Assert on result contents
}

func TestSQLStore_LegalHold_GetChannelIDsForUserDuring(t *testing.T) {
	th := SetupHelper(t).SetupBasic(t)
	defer th.TearDown(t)

	timeReference := mattermostModel.GetMillis()
	startOne := timeReference + 1000000
	endOne := startOne + 10000
	startTwo := startOne + 1000000
	endTwo := startTwo + 10000

	// create a bunch of channels
	channels, err := th.CreateChannels(10, "stale-test", th.User1.Id, th.Team1.Id)
	require.NoError(t, err)

	// Add a bunch of Channel Member History records.
	// 1. In and out before the first search window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[0].Id, startOne-1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[0].Id, startOne-100))

	// 2. In and out before the first window, but then again during the first window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[1].Id, startOne-1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[1].Id, startOne-100))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[1].Id, startOne+1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[1].Id, startOne+2000))

	// 3. In before and out after the first search window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[2].Id, startOne-1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[2].Id, startOne+1000))

	// 4. In before the window and not yet left.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[3].Id, startOne-1000))

	// 5. In during the window and not yet left.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[4].Id, startOne+1000))

	// 6. In after the first window and not yet left.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[5].Id, startTwo-1000))

	// 7. In and out twice during the first window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[6].Id, startOne+1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[6].Id, startOne+2000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[6].Id, startOne+3000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[6].Id, startOne+4000))

	// 8. Leaves at exactly the start of the first window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[7].Id, startOne-1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[7].Id, startOne))

	// 9. Joins at exactly the end of the first window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[8].Id, endOne))

	// 10. Joins during first window, leaves during second window.
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[9].Id, startOne+1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[9].Id, endTwo-1000))

	// Check channel IDs for first window.
	firstWindowChannelIDs, err := th.Store.GetChannelIDsForUserDuring(th.User1.Id, startOne, endOne, false)
	expectedOne := []string{
		channels[1].Id,
		channels[2].Id,
		channels[3].Id,
		channels[4].Id,
		channels[6].Id,
		channels[7].Id,
		channels[9].Id,
	}
	require.NoError(t, err)
	require.ElementsMatch(t, firstWindowChannelIDs, expectedOne)

	// Check channel IDs for second window.
	secondWindowChannelIDs, err := th.Store.GetChannelIDsForUserDuring(th.User1.Id, startTwo, endTwo, false)
	expectedTwo := []string{
		channels[3].Id,
		channels[4].Id,
		channels[5].Id,
		channels[8].Id,
		channels[9].Id,
	}
	require.NoError(t, err)
	require.ElementsMatch(t, secondWindowChannelIDs, expectedTwo)
}

func TestLegalHold_GetChannelIDsForUserDuring_ExcludePublic(t *testing.T) {
	th := SetupHelper(t).SetupBasic(t)
	defer th.TearDown(t)

	timeReference := mattermostModel.GetMillis()
	start := timeReference + 1000000
	end := start + 10000

	openChannel, err := th.CreateChannel("public-channel", th.User1.Id, th.Team1.Id, mattermostModel.ChannelTypeOpen)
	require.NoError(t, err)
	privateChannel, err := th.CreateChannel("private-channel", th.User1.Id, th.Team1.Id, mattermostModel.ChannelTypePrivate)
	require.NoError(t, err)
	dmChannel, err := th.CreateDirectMessageChannel(th.User1, th.User2)
	require.NoError(t, err)
	groupDM, err := th.CreateChannel("group-dm", th.User1.Id, th.Team1.Id, mattermostModel.ChannelTypeGroup)
	require.NoError(t, err)

	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, openChannel.Id, start+1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, privateChannel.Id, start+1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, groupDM.Id, start+1000))
	require.NoError(t, th.mmStore.ChannelMemberHistory().LogJoinEvent(th.User1.Id, dmChannel.Id, start+1000))

	// Check channel IDs
	channelIDs, err := th.Store.GetChannelIDsForUserDuring(th.User1.Id, start, end, true)
	require.NoError(t, err)
	require.ElementsMatch(t, channelIDs, []string{privateChannel.Id, dmChannel.Id, groupDM.Id})
}

func TestSQLStore_LegalHold_GetFileInfosByIDs(t *testing.T) {
	// TODO: Implement me!
	_ = t
}
