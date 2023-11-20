package legalhold

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	LegalHoldID   string
	LegalHoldName string
	StartTime     int64
	EndTime       int64
	UserIDs       []string

	papi        plugin.API
	store       *sqlstore.SQLStore
	fileBackend filestore.FileBackend

	channelIDs []string

	index model.LegalHoldIndex
}

// NewExecution creates a new Execution that is ready to use.
func NewExecution(legalHold model.LegalHold, papi plugin.API, store *sqlstore.SQLStore, fileBackend filestore.FileBackend) Execution {
	return Execution{
		LegalHoldID:   legalHold.ID,
		LegalHoldName: legalHold.Name,
		StartTime:     utils.Max(legalHold.LastExecutionEndedAt, legalHold.StartsAt),
		EndTime:       utils.Min(utils.Max(legalHold.LastExecutionEndedAt, legalHold.StartsAt)+legalHold.ExecutionLength, legalHold.EndsAt),
		UserIDs:       legalHold.UserIDs,
		store:         store,
		fileBackend:   fileBackend,
		index:         make(model.LegalHoldIndex),
		papi:          papi,
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

	return ex.EndTime, nil
}

// GetChannels populates the list of channels that the Execution needs to cover within the
// internal state of the Execution struct.
func (ex *Execution) GetChannels() error {
	for _, userID := range ex.UserIDs {
		user, appErr := ex.papi.GetUser(userID)
		if appErr != nil {
			return appErr
		}

		channelIDs, err := ex.store.GetChannelIDsForUserDuring(userID, ex.StartTime, ex.EndTime)
		if err != nil {
			return err
		}

		ex.channelIDs = append(ex.channelIDs, channelIDs...)

		// Add to channels index
		for _, channelID := range channelIDs {
			if idx, ok := ex.index[userID]; !ok {
				ex.index[userID] = model.LegalHoldIndexUser{
					Username: user.Username,
					Email:    user.Email,
					Channels: []model.LegalHoldChannelMembership{
						{
							ChannelID: channelID,
							StartTime: ex.StartTime,
							EndTime:   ex.EndTime,
						},
					},
				}
			} else {
				ex.index[userID] = model.LegalHoldIndexUser{
					Username: user.Username,
					Email:    user.Email,
					Channels: append(idx.Channels, model.LegalHoldChannelMembership{
						ChannelID: channelID,
						StartTime: ex.StartTime,
						EndTime:   ex.EndTime,
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
		cursor := model.NewLegalHoldCursor(ex.StartTime)
		for {
			var posts []model.LegalHoldPost
			var err error

			posts, cursor, err = ex.store.GetPostsBatch(channelID, ex.EndTime, cursor, PostExportBatchLimit)
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
			return err
		}
	}

	// Write the
	return nil
}

// UpdateIndexes updates the index files in the file backend in relation to this legal hold.
func (ex *Execution) UpdateIndexes() error {
	filePath := ex.channelsIndexPath()

	// Check if the channels index already exists in the file backend.
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
	return err
}

// basePath returns the base file storage path for this Execution.
func (ex *Execution) basePath() string {
	// FIXME: Move onto the LegalHold object, but to do that the LegalHold object needs to be stored
	//        in full in the LegalHold execution, which is a bit more involved.
	return fmt.Sprintf("legal_hold/%s_(%s)", ex.LegalHoldName, ex.LegalHoldID)
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

// channelsIndexPath returns the file path for the Channels Index
// within this Execution.
func (ex *Execution) channelsIndexPath() string {
	return fmt.Sprintf("%s/channels_index.json", ex.basePath())
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
