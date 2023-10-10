package store

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/pkg/errors"
)

// GetPostsBatch fetches a batch of posts from the channel specified by channelID for a legal
// hold export, using the cursor, endTime and limit parameters to track batching state.
// This method was originally based on Mattermost Server's ComplianceStore.ComplianceExport()
// function but considerably simplified to suit the LegalHold use case better.
func (ss *SQLStore) GetPostsBatch(channelID string, endTime int64, cursor model.LegalHoldCursor, limit int) ([]model.LegalHoldPost, model.LegalHoldCursor, error) {
	var posts []model.LegalHoldPost

	var args []any
	// append the named parameters of SQL query in the correct order to args
	args = append(args, cursor.LastPostCreateAt, cursor.LastPostCreateAt, cursor.LastPostID, endTime)
	args = append(args, channelID, limit)

	query := `
		SELECT
			COALESCE(Teams.Name, 'direct-messages') AS TeamName,
			COALESCE(Teams.DisplayName, 'Direct Messages') AS TeamDisplayName,
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
			Posts
		JOIN
		    Users on Posts.UserId = Users.Id
		JOIN
			Channels on Channels.Id = Posts.ChannelId
		LEFT OUTER JOIN
			Bots ON Bots.UserId = Posts.UserId
		LEFT OUTER JOIN
			Teams ON Teams.Id = Channels.TeamId
		WHERE
		 	(
				Posts.CreateAt > ?
				OR (Posts.CreateAt = ? AND Posts.Id > ?)
			)
				AND Posts.CreateAt < ?
			AND Channels.Id = ?
		ORDER BY Posts.CreateAt, Posts.Id
		LIMIT ?
	`

	query = ss.replica.Rebind(query)

	if err := ss.replica.Select(&posts, query, args...); err != nil {
		return nil, cursor, errors.Wrap(err, "unable to get posts batch for legal hold")
	}

	if len(posts) < limit {
		cursor.Completed = true
	} else {
		cursor.LastPostCreateAt = posts[len(posts)-1].PostCreateAt
		cursor.LastPostID = posts[len(posts)-1].PostId
	}

	cursor.BatchNumber += 1

	return append(posts), cursor, nil
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