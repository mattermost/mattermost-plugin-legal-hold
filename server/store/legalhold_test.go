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

func TestSQLStore_LegalholdExport(t *testing.T) {
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
