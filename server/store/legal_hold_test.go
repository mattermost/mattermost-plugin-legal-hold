package store

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-server/v6/model"
)

var (
	yearAgo = model.GetMillisForTime(time.Now().AddDate(-1, 0, 0))
	weekAgo = model.GetMillisForTime(time.Now().AddDate(0, 0, -7))
)

func TestSQLStore_LegalHoldExport(t *testing.T) {
	th := SetupHelper(t).SetupBasic(t)
	defer th.TearDown()

	const channelCount = 10
	const postCount = 10

	// create a bunch of channels
	channels, err := th.CreateChannels(channelCount, "stale-test", th.User1.Id, th.Team1.Id)
	require.NoError(t, err)

	var posts []*model.Post

	// add some posts
	for _, channel := range channels {
		posts, err = th.CreatePosts(postCount, th.User1.Id, channel.Id)
		require.NoError(t, err)
	}

	_ = posts

	compliance := model.Compliance{
		Id:       model.NewId(),
		CreateAt: model.GetMillis(),
		UserId:   th.User1.Id,
		Status:   model.ComplianceStatusCreated,
		Count:    0,
		Desc:     "Description???",
		Type:     model.ComplianceTypeAdhoc,
		StartAt:  model.GetMillis() - 86400000, // Now - 1 day
		EndAt:    model.GetMillis() + 86400000, // Now + 1 day
		Keywords: "",
		Emails:   "",
	}

	cursor := model.ComplianceExportCursor{
		LastChannelsQueryPostCreateAt:       0,
		LastChannelsQueryPostID:             "00000000000000000000000000",
		ChannelsQueryCompleted:              false,
		LastDirectMessagesQueryPostCreateAt: 0,
		LastDirectMessagesQueryPostID:       "00000000000000000000000000",
		DirectMessagesQueryCompleted:        false,
	}

	legalHold, cursor, err := th.Store.LegalholdExport(&compliance, cursor, 1000)
	require.NoError(t, err)
	for _, legalHoldItem := range legalHold {
		t.Log(legalHoldItem)
	}
}

func TestSQLStore_LegalHold_GetChannelIDsForUserDuring(t *testing.T) {
	th := SetupHelper(t).SetupBasic(t)
	defer th.TearDown()

	timeReference := model.GetMillis()
	startOne := timeReference + 1000000
	endOne := startOne + 10000
	startTwo := startOne + 1000000
	endTwo := startTwo + 10000

	// create a bunch of channels
	channels, err := th.CreateChannels(10, "stale-test", th.User1.Id, th.Team1.Id)
	require.NoError(t, err)

	// Add a bunch of Channel Member History records.
	// 1. In and out before the first search window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[0].Id, startOne-1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[0].Id, startOne-100))

	// 2. In and out before the first window, but then again during the first window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[1].Id, startOne-1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[1].Id, startOne-100))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[1].Id, startOne+1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[1].Id, startOne+2000))

	// 3. In before and out after the first search window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[2].Id, startOne-1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[2].Id, startOne+1000))

	// 4. In before the window and not yet left.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[3].Id, startOne-1000))

	// 5. In during the window and not yet left.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[4].Id, startOne+1000))

	// 6. In after the first window and not yet left.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[5].Id, startTwo-1000))

	// 7. In and out twice during the first window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[6].Id, startOne+1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[6].Id, startOne+2000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[6].Id, startOne+3000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[6].Id, startOne+4000))

	// 8. Leaves at exactly the start of the first window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[7].Id, startOne-1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[7].Id, startOne))

	// 9. Joins at exactly the end of the first window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[8].Id, endOne))

	// 10. Joins during first window, leaves during second window.
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogJoinEvent(th.User1.Id, channels[9].Id, startOne+1000))
	require.NoError(t, th.mainHelper.Store.ChannelMemberHistory().LogLeaveEvent(th.User1.Id, channels[9].Id, endTwo-1000))

	// Check channel IDs for first window.
	firstWindowChannelIDs, err := th.Store.GetChannelIDsForUserDuring(th.User1.Id, startOne, endOne)
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
	secondWindowChannelIDs, err := th.Store.GetChannelIDsForUserDuring(th.User1.Id, startTwo, endTwo)
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
