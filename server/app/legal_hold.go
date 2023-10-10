package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gocarina/gocsv"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"strings"
)

const PostExportBatchLimit = 10000

// LegalHoldExecution represents one execution of a LegalHold, i.e. a daily (or other duration)
// batch process to hold all data relating to that particular LegalHold. It is defined by the
// properties of the associated LegalHold as well as a start and end time for the period this
// execution of the LegalHold relates to.
type LegalHoldExecution struct {
	LegalHoldID string
	StartTime   int64
	EndTime     int64
	UserIDs     []string

	store       *store.SQLStore
	fileBackend filestore.FileBackend

	channelIDs []string

	channelsIndex model.LegalHoldChannelIndex
}

// NewLegalHoldExecution creates a new LegalHoldExecution that is ready to use.
func NewLegalHoldExecution(legalHold model.LegalHold, store *store.SQLStore, fileBackend filestore.FileBackend) LegalHoldExecution {
	return LegalHoldExecution{
		LegalHoldID:   legalHold.ID,
		StartTime:     max(legalHold.LastExecutionEndedAt, legalHold.StartsAt),
		EndTime:       min(max(legalHold.LastExecutionEndedAt, legalHold.StartsAt)+legalHold.ExecutionLength, legalHold.EndsAt),
		UserIDs:       legalHold.UserIDs,
		store:         store,
		fileBackend:   fileBackend,
		channelsIndex: make(model.LegalHoldChannelIndex),
	}
}

// Execute executes the LegalHoldExecution.
func (lhe *LegalHoldExecution) Execute() error {
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

// GetChannels populates the list of channels that the LegalHoldExecution needs to cover within the
// internal state of the LegalHoldExecution struct.
func (lhe *LegalHoldExecution) GetChannels() error {
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

	lhe.channelIDs = deduplicateStringSlice(lhe.channelIDs)

	return nil
}

// ExportData is the main function to run the batch data export for this LegalHoldExecution.
func (lhe *LegalHoldExecution) ExportData() error {
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

			err = lhe.WritePostsBatchToFile(cursor.BatchNumber, channelID, posts)
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
func (lhe *LegalHoldExecution) WritePostsBatchToFile(batchNumber uint, channelID string, posts []model.LegalHoldPost) error {
	path := fmt.Sprintf("legal_hold/%s/%s/messages/messages-%d-%s.csv", lhe.LegalHoldID, channelID, posts[0].PostCreateAt, posts[0].PostId)

	csvContent, err := gocsv.MarshalString(&posts)
	if err != nil {
		return err
	}

	csvReader := strings.NewReader(csvContent)

	_, err = lhe.fileBackend.WriteFile(csvReader, path)

	return err
}

// ExportFiles exports the file attachments with the provided FileIDs to the file backend.
func (lhe *LegalHoldExecution) ExportFiles(FileIDs []string) error {
	// TODO: Implement me!
	return nil
}

// UpdateIndexes updates the index files in the file backend in relation to this legal hold.
func (lhe *LegalHoldExecution) UpdateIndexes() error {
	// TODO: Actually read in the old index and merge with the new info.
	// For now, just write out the latest index data as if there isn't any previous data
	// for this legal hold.

	data, err := json.MarshalIndent(lhe.channelsIndex, "", "  ")
	if err != nil {
		return err
	}

	reader := bytes.NewReader(data)

	filePath := fmt.Sprintf("legal_hold/%s/channels_index.json", lhe.LegalHoldID)
	_, err = lhe.fileBackend.WriteFile(reader, filePath)
	return err
}

// deduplicateStringSlice removes duplicate entries from a slice of strings.
func deduplicateStringSlice(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if _, value := keys[item]; !value {
			result = append(result, item)
			keys[item] = true
		}
	}

	return result
}

// max returns the larger of two int64 values.
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two int64 values.
func min(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
