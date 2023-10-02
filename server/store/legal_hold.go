package store

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/pkg/errors"
)

func (ss *SQLStore) LegalholdExport(channel string, endTime int64, cursor model.LegalHoldCursor, limit int) ([]model.LegalHoldPost, model.LegalHoldCursor, error) {
	var channelPosts []model.LegalHoldPost
	channelsQuery := ""
	var argsChannelsQuery []any
	if !cursor.ChannelsQueryCompleted {
		// append the named parameters of SQL query in the correct order to argsChannelsQuery
		argsChannelsQuery = append(argsChannelsQuery, cursor.LastChannelsQueryPostCreateAt, cursor.LastChannelsQueryPostCreateAt, cursor.LastChannelsQueryPostID, endTime)
		argsChannelsQuery = append(argsChannelsQuery, limit)
		channelsQuery = `
		SELECT
			Teams.Name AS TeamName,
			Teams.DisplayName AS TeamDisplayName,
			Channels.Name AS ChannelName,
			Channels.DisplayName AS ChannelDisplayName,
			Channels.Type AS ChannelType,
			Users.Username AS UserUsername,
			Users.Email AS UserEmail,
			Users.Nickname AS UserNickname,
			Posts.Id AS PostId,
			Posts.CreateAt AS PostCreateAt,
			Posts.UpdateAt AS PostUpdateAt,
			Posts.DeleteAt AS PostDeleteAt,
			Posts.RootId AS PostRootId,
			Posts.OriginalId AS PostOriginalId,
			Posts.Message AS PostMessage,
			Posts.Type AS PostType,
			Posts.Props AS PostProps,
			Posts.Hashtags AS PostHashtags,
			Posts.FileIds AS PostFileIds,
			Bots.UserId IS NOT NULL AS IsBot
		FROM
			Teams,
			Channels,
			Users,
			Posts
		LEFT JOIN
			Bots ON Bots.UserId = Posts.UserId
		WHERE
			Teams.Id = Channels.TeamId
				AND Posts.ChannelId = Channels.Id
				AND Posts.UserId = Users.Id
				AND (
					Posts.CreateAt > ?
					OR (Posts.CreateAt = ? AND Posts.Id > ?)
				)
				AND Posts.CreateAt < ?
		ORDER BY Posts.CreateAt, Posts.Id
		LIMIT ?`
		channelsQuery = ss.replica.Rebind(channelsQuery)
		if err := ss.replica.Select(&channelPosts, channelsQuery, argsChannelsQuery...); err != nil {
			return nil, cursor, errors.Wrap(err, "unable to export compliance")
		}
		if len(channelPosts) < limit {
			cursor.ChannelsQueryCompleted = true
		} else {
			cursor.LastChannelsQueryPostCreateAt = channelPosts[len(channelPosts)-1].PostCreateAt
			cursor.LastChannelsQueryPostID = channelPosts[len(channelPosts)-1].PostId
		}
	}

	directMessagePosts := []model.LegalHoldPost{}
	directMessagesQuery := ""
	var argsDirectMessagesQuery []any
	if !cursor.DirectMessagesQueryCompleted && len(channelPosts) < limit {
		// append the named parameters of SQL query in the correct order to argsDirectMessagesQuery
		argsDirectMessagesQuery = append(argsDirectMessagesQuery, cursor.LastDirectMessagesQueryPostCreateAt, cursor.LastDirectMessagesQueryPostCreateAt, cursor.LastDirectMessagesQueryPostID, endTime)
		argsDirectMessagesQuery = append(argsDirectMessagesQuery, limit-len(channelPosts))
		directMessagesQuery = `
		SELECT
			'direct-messages' AS TeamName,
			'Direct Messages' AS TeamDisplayName,
			Channels.Name AS ChannelName,
			Channels.DisplayName AS ChannelDisplayName,
			Channels.Type AS ChannelType,
			Users.Username AS UserUsername,
			Users.Email AS UserEmail,
			Users.Nickname AS UserNickname,
			Posts.Id AS PostId,
			Posts.CreateAt AS PostCreateAt,
			Posts.UpdateAt AS PostUpdateAt,
			Posts.DeleteAt AS PostDeleteAt,
			Posts.RootId AS PostRootId,
			Posts.OriginalId AS PostOriginalId,
			Posts.Message AS PostMessage,
			Posts.Type AS PostType,
			Posts.Props AS PostProps,
			Posts.Hashtags AS PostHashtags,
			Posts.FileIds AS PostFileIds,
			Bots.UserId IS NOT NULL AS IsBot
		FROM
			Channels,
			Users,
			Posts
		LEFT JOIN
			Bots ON Bots.UserId = Posts.UserId
		WHERE
			Channels.TeamId = ''
				AND Posts.ChannelId = Channels.Id
				AND Posts.UserId = Users.Id
				AND (
					Posts.CreateAt > ?
					OR (Posts.CreateAt = ? AND Posts.Id > ?)
				)
				AND Posts.CreateAt < ?
		ORDER BY Posts.CreateAt, Posts.Id
		LIMIT ?`

		directMessagesQuery = ss.replica.Rebind(directMessagesQuery)
		if err := ss.replica.Select(&directMessagePosts, directMessagesQuery, argsDirectMessagesQuery...); err != nil {
			return nil, cursor, errors.Wrap(err, "unable to export compliance")
		}
		if len(directMessagePosts) < limit {
			cursor.DirectMessagesQueryCompleted = true
		} else {
			cursor.LastDirectMessagesQueryPostCreateAt = directMessagePosts[len(directMessagePosts)-1].PostCreateAt
			cursor.LastDirectMessagesQueryPostID = directMessagePosts[len(directMessagePosts)-1].PostId
		}
	}

	return append(channelPosts, directMessagePosts...), cursor, nil
}

// GetChannelIDsForUserDuring gets the channel IDs for all channels that the user indicated by userID is
// a member of during the time period from (and including) the startTime up until (but not including) the
// endTime.
func (ss *SQLStore) GetChannelIDsForUserDuring(userID string, startTime int64, endTime int64) ([]string, error) {
	query := ss.replicaBuilder.
		Select("distinct(cmh.channelid)").
		From("channelmemberhistory as cmh").
		Where(sq.Lt{"cmh.jointime": endTime}).
		Where(sq.Or{sq.Eq{"cmh.leavetime": nil}, sq.GtOrEq{"cmh.leavetime": startTime}}).
		Where(sq.Eq{"cmh.userid": userID})

	rows, err := query.Query()
	if err != nil {
		ss.logger.Error("error fetching channels for user during time period", "err", err)
		return []string{}, err
	}

	var channelIDs []string
	for rows.Next() {
		var channelID string

		if err := rows.Scan(&channelID); err != nil {
			ss.logger.Error("error scanning channel of channels for user during time period", "err", err)
			return []string{}, err
		}
		channelIDs = append(channelIDs, channelID)
	}

	return channelIDs, nil
}
