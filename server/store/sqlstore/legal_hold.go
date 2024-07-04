package sqlstore

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	mattermostModel "github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

// GetPostsBatch fetches a batch of posts from the channel specified by channelID for a legal
// hold export, using the cursor, endTime and limit parameters to track batching state.
// This method was originally based on Mattermost Server's ComplianceStore.ComplianceExport()
// function but considerably simplified to suit the LegalHold use case better.
func (ss SQLStore) GetPostsBatch(channelID string, endTime int64, cursor model.LegalHoldCursor, limit int) ([]model.LegalHoldPost, model.LegalHoldCursor, error) {
	var posts []model.LegalHoldPost

	var args []any
	// append the named parameters of SQL query in the correct order to args
	args = append(args, cursor.LastPostCreateAt, cursor.LastPostCreateAt, cursor.LastPostID, endTime)
	args = append(args, channelID, limit)

	dmDisplayName := `
						(select Users.Username from Users where Users.Id = split_part(Channels.Name, '__', 1))
							|| ', ' ||
						(select Users.Username from Users where Users.Id = split_part(Channels.Name, '__', 2))
						`

	if ss.src.DriverName() == mattermostModel.DatabaseDriverMysql {
		dmDisplayName = `
						concat(
							(select Users.Username from Users where Users.Id = substring_index(Channels.Name, '__', 1)),
							', ',
							(select Users.Username from Users where Users.Id = substring_index(Channels.Name, '__', -1))
						)
						`
	}

	query := `
		SELECT
			COALESCE(Teams.Name, 'direct-messages') AS TeamName,
			COALESCE(Teams.DisplayName, 'Direct Messages') AS TeamDisplayName,
			Channels.Name AS ChannelName,
			CASE
				WHEN Channels.Type = 'D' THEN
					(
						` + dmDisplayName + `
					)
				ELSE
					Channels.DisplayName
				END
				AS ChannelDisplayName,
			Channels.Type AS ChannelType,
			Users.Username AS UserUsername,
			Users.Email AS UserEmail,
			Users.Nickname AS UserNickname,
			Posts.ID AS PostID,
			Posts.CreateAt AS PostCreateAt,
			Posts.UpdateAt AS PostUpdateAt,
			Posts.DeleteAt AS PostDeleteAt,
			Posts.RootId AS PostRootID,
			Posts.OriginalId AS PostOriginalID,
			Posts.Message AS PostMessage,
			Posts.Type AS PostType,
			Posts.Props AS PostProps,
			Posts.Hashtags AS PostHashtags,
			Posts.FileIds AS PostFileIDs,
			Bots.UserId IS NOT NULL AS IsBot
		FROM
			Posts
		JOIN
		    Users on Posts.UserId = Users.ID
		JOIN
			Channels on Channels.ID = Posts.ChannelId
		LEFT OUTER JOIN
			Bots ON Bots.UserId = Posts.UserId
		LEFT OUTER JOIN
			Teams ON Teams.ID = Channels.TeamId
		WHERE
		 	(
				Posts.CreateAt > ?
				OR (Posts.CreateAt = ? AND Posts.ID > ?)
			)
				AND Posts.CreateAt < ?
			AND Channels.ID = ?
		ORDER BY Posts.CreateAt, Posts.ID
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
		cursor.LastPostID = posts[len(posts)-1].PostID
	}

	cursor.BatchNumber++

	return posts, cursor, nil
}

// GetChannelIDsForUserDuring gets the channel IDs for all channels that the user indicated by userID is
// a member of during the time period from (and including) the startTime up until (but not including) the
// endTime.
func (ss SQLStore) GetChannelIDsForUserDuring(userID string, startTime int64, endTime int64, excludePublic bool) ([]string, error) {
	query := ss.replicaBuilder.
		Select("distinct(cmh.channelid)").
		From("ChannelMemberHistory as cmh").
		Where(sq.Lt{"cmh.jointime": endTime}).
		Where(sq.Or{sq.Eq{"cmh.leavetime": nil}, sq.GtOrEq{"cmh.leavetime": startTime}}).
		Where(sq.Eq{"cmh.userid": userID})

	// Exclude all public channels from the results
	if excludePublic {
		query = query.Join("channels on cmh.channelid = channels.id").
			Where(sq.NotEq{"channels.type": mattermostModel.ChannelTypeOpen})
	}

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

// GetFileInfosByIDs gets the file infos corresponding to the provided ids.
func (ss SQLStore) GetFileInfosByIDs(ids []string) ([]model.FileInfo, error) {
	query := ss.replicaBuilder.
		Select(
			"FileInfo.ID",
			"FileInfo.Path",
			"FileInfo.Name",
			"FileInfo.Size",
			"FileInfo.MimeType",
		).
		From("FileInfo").
		Where(sq.Eq{"FileInfo.ID": ids}).
		OrderBy("FileInfo.CreateAt DESC")

	sql, args, err := query.ToSql()
	if err != nil {
		return []model.FileInfo{}, errors.Wrap(err, "unable to get sql for GetFileInfosByIDs")
	}

	var fileInfos []model.FileInfo
	err = ss.replica.Select(&fileInfos, sql, args...)
	if err != nil {
		return []model.FileInfo{}, errors.Wrap(err, "unable to run query for GetFileInfosByIDs")
	}

	return fileInfos, nil
}

// GetChannelMetadataForIDs receives a list of channelIDs and returns the ChannelMetadata for each
// of the identified channels. ChannelMetadata is all the additional data that is needed to populate
// the Legal Hold index file with information about the channel.
func (ss SQLStore) GetChannelMetadataForIDs(channelIDs []string) ([]model.ChannelMetadata, error) {
	var data []model.ChannelMetadata

	query := `
		SELECT
			COALESCE(Teams.Id, '00000000000000000000000000') AS TeamID,
			COALESCE(Teams.Name, 'direct-messages') AS TeamName,
			COALESCE(Teams.DisplayName, 'Direct Messages') AS TeamDisplayName,
			Channels.Id as ChannelID,
			Channels.Name AS ChannelName,
			Channels.Type AS ChannelType,
			CASE
				WHEN Channels.Type = 'D' THEN
					(
						(select Users.Username from Users where Users.Id = split_part(Channels.Name, '__', 1))
							|| ', ' ||
						(select Users.Username from Users where Users.Id = split_part(Channels.Name, '__', 2))
					)
				ELSE
					Channels.DisplayName
				END
				AS ChannelDisplayName
		FROM
			Channels
		LEFT OUTER JOIN
			Teams ON Teams.ID = Channels.TeamId
		WHERE
			Channels.Id IN (?)`

	query, args, err := sqlx.In(query, channelIDs)
	if err != nil {
		return nil, errors.Wrap(err, "unable to get channel metadata")
	}
	query = ss.replica.Rebind(query)

	if err := ss.replica.Select(&data, query, args...); err != nil {
		return nil, errors.Wrap(err, "unable to get channel metadata")
	}

	return data, nil
}
