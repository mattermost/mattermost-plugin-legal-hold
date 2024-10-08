package legalhold

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"
)

const PostExportBatchLimit = 10000

// Execution represents one execution of a LegalHold, i.e. a daily (or other duration)
// batch process to hold all data relating to that particular LegalHold. It is defined by the
// properties of the associated LegalHold as well as a start and end time for the period this
// execution of the LegalHold relates to.
type Execution struct {
	LegalHold          model.LegalHold
	ExecutionStartTime int64
	ExecutionEndTime   int64

	papi        plugin.API
	store       *sqlstore.SQLStore
	fileBackend filestore.FileBackend

	channelIDs []string

	index  model.LegalHoldIndex
	hashes model.HashList
}

// NewExecution creates a new Execution that is ready to use.
func NewExecution(legalHold model.LegalHold, papi plugin.API, store *sqlstore.SQLStore, fileBackend filestore.FileBackend) Execution {
	return Execution{
		LegalHold:          legalHold,
		ExecutionStartTime: legalHold.NextExecutionStartTime(),
		ExecutionEndTime:   legalHold.NextExecutionEndTime(),
		store:              store,
		fileBackend:        fileBackend,
		index:              model.NewLegalHoldIndex(),
		papi:               papi,
		hashes:             make(map[string]string),
	}
}

// Execute executes the Execution.
func (ex *Execution) Execute() (int64, error) {
	err := ex.GetChannels()
	if err != nil {
		return 0, err
	}

	err = ex.ExportData()
	if err != nil {
		return 0, err
	}

	err = ex.UpdateIndexes()
	if err != nil {
		return 0, err
	}

	err = ex.WriteFileHashes()
	if err != nil {
		return 0, err
	}

	return ex.ExecutionEndTime, nil
}

// GetChannels populates the list of channels that the Execution needs to cover within the
// internal state of the Execution struct.
func (ex *Execution) GetChannels() error {
	for _, userID := range ex.LegalHold.UserIDs {
		user, appErr := ex.papi.GetUser(userID)
		if appErr != nil {
			return appErr
		}

		channelIDs, err := ex.store.GetChannelIDsForUserDuring(userID, ex.ExecutionStartTime, ex.ExecutionEndTime, ex.LegalHold.IncludePublicChannels)
		if err != nil {
			return err
		}

		ex.channelIDs = append(ex.channelIDs, channelIDs...)

		// Add to channels index
		for _, channelID := range channelIDs {
			if idx, ok := ex.index.Users[userID]; !ok {
				ex.index.Users[userID] = model.LegalHoldIndexUser{
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
				ex.index.Users[userID] = model.LegalHoldIndexUser{
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

			err = ex.WritePostsBatchToFile(channelID, posts)
			if err != nil {
				return err
			}

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
func (ex *Execution) UpdateIndexes() error {
	filePath := ex.indexPath()

	// Populate the metadata in the index.
	ex.index.LegalHold.ID = ex.LegalHold.ID
	ex.index.LegalHold.DisplayName = ex.LegalHold.DisplayName
	ex.index.LegalHold.Name = ex.LegalHold.Name
	ex.index.LegalHold.StartsAt = ex.LegalHold.StartsAt
	ex.index.LegalHold.LastExecutionEndedAt = ex.ExecutionEndTime

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
	return fmt.Sprintf("%s/index.json", ex.basePath())
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
