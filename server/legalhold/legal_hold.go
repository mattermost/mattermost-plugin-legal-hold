package legalhold

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store/sqlstore"
	"strings"

	"github.com/gocarina/gocsv"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/utils"
)

const PostExportBatchLimit = 10000

// Execution represents one execution of a LegalHold, i.e. a daily (or other duration)
// batch process to hold all data relating to that particular LegalHold. It is defined by the
// properties of the associated LegalHold as well as a start and end time for the period this
// execution of the LegalHold relates to.
type Execution struct {
	LegalHoldID   string
	LegalHoldName string
	StartTime     int64
	EndTime       int64
	UserIDs       []string

	store       *sqlstore.SQLStore
	fileBackend filestore.FileBackend

	channelIDs []string

	channelsIndex model.LegalHoldChannelIndex
}

// NewExecution creates a new Execution that is ready to use.
func NewExecution(legalHold model.LegalHold, store *sqlstore.SQLStore, fileBackend filestore.FileBackend) Execution {
	return Execution{
		LegalHoldID:   legalHold.ID,
		LegalHoldName: legalHold.Name,
		StartTime:     utils.Max(legalHold.LastExecutionEndedAt, legalHold.StartsAt),
		EndTime:       utils.Min(utils.Max(legalHold.LastExecutionEndedAt, legalHold.StartsAt)+legalHold.ExecutionLength, legalHold.EndsAt),
		UserIDs:       legalHold.UserIDs,
		store:         store,
		fileBackend:   fileBackend,
		channelsIndex: make(model.LegalHoldChannelIndex),
	}
}

// Execute executes the Execution.
func (lhe *Execution) Execute() error {
	err := lhe.GetChannels()
	if err != nil {
		return err
	}

	err = lhe.ExportData()
	if err != nil {
		return err
	}

	err = lhe.UpdateIndexes()
	return err
}

// GetChannels populates the list of channels that the Execution needs to cover within the
// internal state of the Execution struct.
func (lhe *Execution) GetChannels() error {
	for _, userID := range lhe.UserIDs {
		channelIDs, err := lhe.store.GetChannelIDsForUserDuring(userID, lhe.StartTime, lhe.EndTime)
		if err != nil {
			return err
		}

		lhe.channelIDs = append(lhe.channelIDs, channelIDs...)

		// Add to channels index
		for _, channelID := range channelIDs {
			if idx, ok := lhe.channelsIndex[userID]; !ok {
				lhe.channelsIndex[userID] = []model.LegalHoldChannelMembership{
					{
						ChannelID: channelID,
						StartTime: lhe.StartTime,
						EndTime:   lhe.EndTime,
					},
				}
			} else {
				lhe.channelsIndex[userID] = append(idx, model.LegalHoldChannelMembership{
					ChannelID: channelID,
					StartTime: lhe.StartTime,
					EndTime:   lhe.EndTime,
				})
			}
		}
	}

	lhe.channelIDs = utils.DeduplicateStringSlice(lhe.channelIDs)

	return nil
}

// ExportData is the main function to run the batch data export for this Execution.
func (lhe *Execution) ExportData() error {
	for _, channelID := range lhe.channelIDs {
		cursor := model.NewLegalHoldCursor(lhe.StartTime)
		for {
			var posts []model.LegalHoldPost
			var err error

			posts, cursor, err = lhe.store.GetPostsBatch(channelID, lhe.EndTime, cursor, PostExportBatchLimit)
			if err != nil {
				return err
			}

			if len(posts) == 0 {
				break
			}

			err = lhe.WritePostsBatchToFile(channelID, posts)
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

			err = lhe.ExportFiles(channelID, posts[0].PostCreateAt, posts[0].PostID, fileIDs)
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
func (lhe *Execution) WritePostsBatchToFile(channelID string, posts []model.LegalHoldPost) error {
	path := lhe.messagesBatchPath(channelID, posts[0].PostCreateAt, posts[0].PostID)

	csvContent, err := gocsv.MarshalString(&posts)
	if err != nil {
		return err
	}

	csvReader := strings.NewReader(csvContent)

	_, err = lhe.fileBackend.WriteFile(csvReader, path)

	return err
}

// ExportFiles exports the file attachments with the provided FileIDs to the file backend.
func (lhe *Execution) ExportFiles(channelID string, batchCreateAt int64, batchPostID string, fileIDs []string) error {
	if len(fileIDs) == 0 {
		return nil
	}

	// Batch get the FileInfos for the FileIDs.
	fileInfos, err := lhe.store.GetFileInfosByIDs(fileIDs)
	if err != nil {
		return err
	}

	// Copy the files from one to another.
	for _, fileInfo := range fileInfos {
		path := lhe.filePath(
			channelID,
			batchCreateAt,
			batchPostID,
			fileInfo.ID,
			fileInfo.Name,
		)
		err = lhe.fileBackend.CopyFile(fileInfo.Path, path)
		if err != nil {
			return err
		}
	}

	// Write the
	return nil
}

// UpdateIndexes updates the index files in the file backend in relation to this legal hold.
func (lhe *Execution) UpdateIndexes() error {
	filePath := lhe.channelsIndexPath()

	// Check if the channels index already exists in the file backend.
	if exists, err := lhe.fileBackend.FileExists(filePath); err != nil {
		return err
	} else if exists {
		// Index already exists. Need to read it and then merge with the new data.
		readData, err := lhe.fileBackend.ReadFile(filePath)
		if err != nil {
			return err
		}

		var existingIndex model.LegalHoldChannelIndex
		err = json.Unmarshal(readData, &existingIndex)
		if err != nil {
			return err
		}

		existingIndex.Merge(&lhe.channelsIndex)
		lhe.channelsIndex = existingIndex
	}

	// Write the index data out to the file backend.
	data, err := json.MarshalIndent(lhe.channelsIndex, "", "  ")
	if err != nil {
		return err
	}

	reader := bytes.NewReader(data)

	_, err = lhe.fileBackend.WriteFile(reader, filePath)
	return err
}

// basePath returns the base file storage path for this Execution.
func (lhe *Execution) basePath() string {
	return fmt.Sprintf("legal_hold/%s_(%s)", lhe.LegalHoldName, lhe.LegalHoldID)
}

// channelPath returns the base file storage path for a given channel within
// this Execution.
func (lhe *Execution) channelPath(channelID string) string {
	return fmt.Sprintf("%s/%s", lhe.basePath(), channelID)
}

// messageBatchPath returns the file path for a given message batch
// within this Execution.
func (lhe *Execution) messagesBatchPath(channelID string, batchCreateAt int64, batchPostID string) string {
	return fmt.Sprintf(
		"%s/messages/messages-%d-%s.csv",
		lhe.channelPath(channelID),
		batchCreateAt,
		batchPostID,
	)
}

// channelsIndexPath returns the file path for the Channels Index
// within this Execution.
func (lhe *Execution) channelsIndexPath() string {
	return fmt.Sprintf("%s/channels_index.json", lhe.basePath())
}

// filePath returns the file path for a given file attachment within
// this Execution.
func (lhe *Execution) filePath(channelID string, batchCreateAt int64, batchPostID string, fileID string, fileName string) string {
	return fmt.Sprintf(
		"%s/files/files-%d-%s/%s/%s",
		lhe.channelPath(channelID),
		batchCreateAt,
		batchPostID,
		fileID,
		fileName,
	)
}
