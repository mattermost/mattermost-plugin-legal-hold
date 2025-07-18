package legalhold

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/mattermost/mattermost-plugin-api/cluster"
	mm_model "github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/kvstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"
)

const PostExportBatchLimit = 10000

// executionWaitForLockTimeout is the time to wait for the lock to be acquired before failing the execution.
// This is to prevent multiple executions at the same time.
const executionWaitForLockTimeout = 5 * time.Second

// Execution represents one execution of a LegalHold, i.e. a daily (or other duration)
// batch process to hold all data relating to that particular LegalHold. It is defined by the
// properties of the associated LegalHold as well as a start and end time for the period this
// execution of the LegalHold relates to.
type Execution struct {
	LegalHold          *model.LegalHold
	ExecutionStartTime int64
	ExecutionEndTime   int64

	papi        plugin.API
	store       *sqlstore.SQLStore
	kvstore     kvstore.KVStore
	fileBackend filestore.FileBackend

	channelIDs []string

	index  model.LegalHoldIndex
	hashes model.HashList
}

// NewExecution creates a new Execution that is ready to use.
func NewExecution(legalHold model.LegalHold, papi plugin.API, store *sqlstore.SQLStore, kvstore kvstore.KVStore, fileBackend filestore.FileBackend) Execution {
	return Execution{
		LegalHold:          &legalHold,
		ExecutionStartTime: legalHold.NextExecutionStartTime(),
		ExecutionEndTime:   legalHold.NextExecutionEndTime(),
		store:              store,
		kvstore:            kvstore,
		fileBackend:        fileBackend,
		index:              model.NewLegalHoldIndex(),
		papi:               papi,
		hashes:             make(map[string]string),
	}
}

// Execute executes the Execution and returns the updated LegalHold.
func (ex *Execution) Execute(now int64) (*model.LegalHold, error) {
	// Lock multiple executions behind a cluster mutex
	mutex, err := cluster.NewMutex(ex.papi, "legal_hold_execution")
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster mutex: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), executionWaitForLockTimeout)
	defer cancel()

	if lockErr := mutex.LockWithContext(ctx); lockErr != nil {
		return nil, fmt.Errorf("failed to lock cluster mutex: %w", lockErr)
	}
	defer func() {
		mutex.Unlock()
	}()

	err = ex.GetChannels()
	if err != nil {
		return nil, err
	}

	err = ex.ExportData()
	if err != nil {
		return nil, err
	}

	err = ex.UpdateIndexes(now)
	if err != nil {
		return nil, err
	}

	err = ex.WriteFileHashes()
	if err != nil {
		return nil, err
	}

	// Update the LegalHold with execution results
	// Ensure that the LastExecutionEndedAt is not in the future, useful when running the job manually
	ex.LegalHold.LastExecutionEndedAt = utils.Min(ex.ExecutionEndTime, now)

	return ex.LegalHold, nil
}

// GetChannels populates the list of channels that the Execution needs to cover within the
// internal state of the Execution struct.
func (ex *Execution) GetChannels() error {
	targetUsers, appErr := getUsersForGroups(ex.papi, ex.LegalHold.GroupIDs)
	if appErr != nil {
		return appErr
	}

	for _, userID := range ex.LegalHold.UserIDs {
		user, appErr := ex.papi.GetUser(userID)
		if appErr != nil {
			return appErr
		}
		targetUsers = append(targetUsers, user)
	}

	// keep track of which users have been processed
	processedUsersList := make(map[string]struct{})
	// processAndMarkUser is a helper function that will check if a user
	// has been processed, and mark the user if they has not.
	// Returns true if the user should be processed
	processAndMarkUser := func(id string) bool {
		if _, processed := processedUsersList[id]; processed {
			return false
		}
		processedUsersList[id] = struct{}{}
		return true
	}

	for _, user := range targetUsers {
		if !processAndMarkUser(user.Id) {
			continue
		}

		channelIDs, err := ex.store.GetChannelIDsForUserDuring(user.Id, ex.ExecutionStartTime, ex.ExecutionEndTime, ex.LegalHold.IncludePublicChannels)
		if err != nil {
			return err
		}

		ex.channelIDs = append(ex.channelIDs, channelIDs...)

		ex.papi.LogDebug(
			"Legal hold executor - GetChannels",
			"user_id", user.Id,
			"channel_count", len(channelIDs),
			"start_time", ex.ExecutionStartTime,
			"end_time", ex.ExecutionEndTime,
		)

		// Add to channels index
		for _, channelID := range channelIDs {
			if idx, ok := ex.index.Users[user.Id]; !ok {
				ex.index.Users[user.Id] = model.LegalHoldIndexUser{
					Username: user.Username,
					Email:    user.Email,
					Channels: []model.LegalHoldChannelMembership{
						{
							ChannelID: channelID,
							StartTime: ex.ExecutionStartTime,
							EndTime:   ex.ExecutionEndTime,
						},
					},
				}
			} else {
				ex.index.Users[user.Id] = model.LegalHoldIndexUser{
					Username: user.Username,
					Email:    user.Email,
					Channels: append(idx.Channels, model.LegalHoldChannelMembership{
						ChannelID: channelID,
						StartTime: ex.ExecutionStartTime,
						EndTime:   ex.ExecutionEndTime,
					}),
				}
			}
		}
	}

	ex.channelIDs = utils.DeduplicateStringSlice(ex.channelIDs)

	return nil
}

// ExportData is the main function to run the batch data export for this Execution.
func (ex *Execution) ExportData() error {
	for _, channelID := range ex.channelIDs {
		cursor := model.NewLegalHoldCursor(ex.ExecutionStartTime)
		for {
			var posts []model.LegalHoldPost
			var err error

			posts, cursor, err = ex.store.GetPostsBatch(channelID, ex.ExecutionEndTime, cursor, PostExportBatchLimit)
			if err != nil {
				return err
			}

			if len(posts) == 0 {
				break
			}

			ex.papi.LogDebug("Legal hold executor - ExportData", "channel_id", channelID, "post_count", len(posts))

			err = ex.WritePostsBatchToFile(channelID, posts)
			if err != nil {
				return err
			}

			// Since at this point we have posts, ensure the `HasMessages` is set to true so users can
			// download the legal hold.
			ex.LegalHold.HasMessages = true

			// Extract the FileIDs to export
			var fileIDs []string
			for _, post := range posts {
				var postFileIDs []string
				err = json.Unmarshal([]byte(post.PostFileIDs), &postFileIDs)
				if err != nil {
					return err
				}
				fileIDs = append(fileIDs, postFileIDs...)
			}

			ex.papi.LogDebug("Legal hold executor - ExportData", "channel_id", channelID, "file_count", len(fileIDs))

			err = ex.ExportFiles(channelID, posts[0].PostCreateAt, posts[0].PostID, fileIDs)
			if err != nil {
				return err
			}

			if len(posts) < PostExportBatchLimit {
				break
			}
		}
	}

	return nil
}

// WritePostsBatchToFile writes a batch of posts from a channel to the appropriate file
// in the file backend.
func (ex *Execution) WritePostsBatchToFile(channelID string, posts []model.LegalHoldPost) error {
	path := ex.messagesBatchPath(channelID, posts[0].PostCreateAt, posts[0].PostID)

	csvContent, err := gocsv.MarshalString(&posts)
	if err != nil {
		return err
	}

	csvReader := strings.NewReader(csvContent)

	_, err = ex.fileBackend.WriteFile(csvReader, path)
	if err != nil {
		return err
	}

	hashReader := strings.NewReader(csvContent)

	h, err := hashFromReader(ex.LegalHold.Secret, hashReader)
	if err != nil {
		return err
	}

	err = ex.WriteFileHash(path, h)

	return err
}

// ExportFiles exports the file attachments with the provided FileIDs to the file backend.
func (ex *Execution) ExportFiles(channelID string, batchCreateAt int64, batchPostID string, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	// Batch get the FileInfos for the FileIDs.
	fileInfos, err := ex.store.GetFileInfosByIDs(fileIDs)
	if err != nil {
		return err
	}

	// Copy the files from one to another.
	for _, fileInfo := range fileInfos {
		path := ex.filePath(
			channelID,
			batchCreateAt,
			batchPostID,
			fileInfo.ID,
			fileInfo.Name,
		)
		err = ex.fileBackend.CopyFile(fileInfo.Path, path)
		if err != nil {
			ex.papi.LogError(fmt.Sprintf("Failed to find file attachment to copy %s", fileInfo.Path))
			// Continue anyway so the job doesn't get completely stuck.
			return nil
		}

		hashReader, err := ex.fileBackend.Reader(fileInfo.Path)
		if err != nil {
			return err
		}

		h, err := hashFromReader(ex.LegalHold.Secret, hashReader)
		if err != nil {
			return err
		}

		err = ex.WriteFileHash(path, h)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateIndexes updates the index files in the file backend in relation to this legal hold.
func (ex *Execution) UpdateIndexes(now int64) error {
	filePath := ex.indexPath()

	// Populate the metadata in the index.
	ex.index.LegalHold.ID = ex.LegalHold.ID
	ex.index.LegalHold.DisplayName = ex.LegalHold.DisplayName
	ex.index.LegalHold.Name = ex.LegalHold.Name
	ex.index.LegalHold.StartsAt = ex.LegalHold.StartsAt
	ex.index.LegalHold.LastExecutionEndedAt = utils.Min(ex.ExecutionEndTime, now)

	if len(ex.channelIDs) > 0 {
		metadata, err := ex.store.GetChannelMetadataForIDs(ex.channelIDs)
		if err != nil {
			return err
		}

		for _, m := range metadata {
			foundTeam := false
			for _, t := range ex.index.Teams {
				if t.ID == m.TeamID {
					foundTeam = true
					t.Channels = append(t.Channels, &model.LegalHoldChannel{
						ID:          m.ChannelID,
						Name:        m.ChannelName,
						DisplayName: m.ChannelDisplayName,
						Type:        m.ChannelType,
					})
					break
				}
			}

			if !foundTeam {
				ex.index.Teams = append(ex.index.Teams, &model.LegalHoldTeam{
					ID:          m.TeamID,
					Name:        m.TeamName,
					DisplayName: m.TeamDisplayName,
					Channels: []*model.LegalHoldChannel{
						{
							ID:          m.ChannelID,
							Name:        m.ChannelName,
							DisplayName: m.ChannelDisplayName,
							Type:        m.ChannelType,
						},
					},
				})
			}
		}
	}

	// Check if the index already exists in the file backend.
	if exists, err := ex.fileBackend.FileExists(filePath); err != nil {
		return err
	} else if exists {
		// Index already exists. Need to read it and then merge with the new data.
		readData, err := ex.fileBackend.ReadFile(filePath)
		if err != nil {
			return err
		}

		var existingIndex model.LegalHoldIndex
		err = json.Unmarshal(readData, &existingIndex)
		if err != nil {
			return err
		}

		existingIndex.Merge(&ex.index)
		ex.index = existingIndex
	}

	// Write the index data out to the file backend.
	data, err := json.MarshalIndent(ex.index, "", "  ")
	if err != nil {
		return err
	}

	reader := bytes.NewReader(data)

	_, err = ex.fileBackend.WriteFile(reader, filePath)
	if err != nil {
		return err
	}

	hashReader := bytes.NewReader(data)

	h, err := hashFromReader(ex.LegalHold.Secret, hashReader)
	if err != nil {
		return err
	}

	err = ex.WriteFileHash(filePath, h)

	return err
}

func (ex *Execution) WriteFileHash(path, hash string) error {
	ex.hashes[path] = hash
	return nil
}

func (ex *Execution) WriteFileHashes() error {
	hashesFilePath := fmt.Sprintf("%s/hashes.json", ex.basePath())

	if exists, err := ex.fileBackend.FileExists(hashesFilePath); err != nil {
		return fmt.Errorf("failed to check if hashes file exists: %w", err)
	} else if !exists {
		// If the file does not exist, just write the hashes we have into it

		hashesFileContent, err := json.MarshalIndent(ex.hashes, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal hashes: %w", err)
		}

		_, err = ex.fileBackend.WriteFile(bytes.NewReader(hashesFileContent), hashesFilePath)
		if err != nil {
			return err
		}
	} else {
		data, err := ex.fileBackend.ReadFile(hashesFilePath)
		if err != nil {
			return fmt.Errorf("failed to open hashes.json file: %w", err)
		}

		var currentHashes model.HashList
		err = json.Unmarshal(data, &currentHashes)
		if err != nil {
			return fmt.Errorf("failed to unmarshal hashes.json file: %w", err)
		}
		for path, hash := range ex.hashes {
			currentHashes[path] = hash
		}

		// Write the updated hashes to the file
		hashesFileContent, err := json.MarshalIndent(currentHashes, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal updated hashes: %w", err)
		}

		lineReader := bytes.NewReader(hashesFileContent)

		_, err = ex.fileBackend.WriteFile(lineReader, hashesFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// basePath returns the base file storage path for this Execution.
func (ex *Execution) basePath() string {
	return ex.LegalHold.BasePath()
}

// channelPath returns the base file storage path for a given channel within
// this Execution.
func (ex *Execution) channelPath(channelID string) string {
	return fmt.Sprintf("%s/%s", ex.basePath(), channelID)
}

// messageBatchPath returns the file path for a given message batch
// within this Execution.
func (ex *Execution) messagesBatchPath(channelID string, batchCreateAt int64, batchPostID string) string {
	return fmt.Sprintf(
		"%s/messages/messages-%d-%s.csv",
		ex.channelPath(channelID),
		batchCreateAt,
		batchPostID,
	)
}

// indexPath returns the file path for the Index file for this LegalHold.
func (ex *Execution) indexPath() string {
	return ex.LegalHold.IndexPath()
}

// filePath returns the file path for a given file attachment within
// this Execution.
func (ex *Execution) filePath(channelID string, batchCreateAt int64, batchPostID string, fileID string, fileName string) string {
	return fmt.Sprintf(
		"%s/files/files-%d-%s/%s/%s",
		ex.channelPath(channelID),
		batchCreateAt,
		batchPostID,
		fileID,
		fileName,
	)
}

// hashFromReader returns the HMAC-SHA512 hash of the reader's contents.
func hashFromReader(secret string, reader io.Reader) (string, error) {
	hasher := hmac.New(sha512.New, []byte(secret))

	_, err := io.Copy(hasher, reader)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func getUsersForGroups(api plugin.API, groupIDs []string) ([]*mm_model.User, error) {
	const GroupPageLimit = 100
	const GroupPageSize = 50

	var allUsers []*mm_model.User
	for _, groupID := range groupIDs {
		currPage := 0
		for {
			users, appErr := api.GetGroupMemberUsers(groupID, currPage, GroupPageSize)
			if appErr != nil {
				return nil, appErr
			}
			if currPage > GroupPageLimit {
				return nil, fmt.Errorf("cannot execute legal hold: a group (%s) exceeds the maximum number of members (%d)", groupID, GroupPageLimit*GroupPageSize)
			}
			if len(users) < 1 {
				break
			}
			allUsers = append(allUsers, users...)
			currPage++
		}
	}

	return allUsers, nil
}
