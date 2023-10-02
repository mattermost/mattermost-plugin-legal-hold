package app

import (
	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
	"github.com/mattermost/mattermost-plugin-legal-hold/server/store"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
)

const POST_EXPORT_BATCH_LIMIT = 10000

type LegalHoldExecution struct {
	LegalHoldID string
	StartTime   int64
	EndTime     int64
	UserIDs     []string

	store     *store.SQLStore
	filestore *filestore.FileBackend

	channelIDs []string
}

// GetChannels populates the list of channels that the Legal Hold Export needs to cover within the
// internal state of the LegalHoldExecution struct.
func (lhe *LegalHoldExecution) GetChannels() error {
	for _, userID := range lhe.UserIDs {
		channelIDs, err := lhe.store.GetChannelIDsForUserDuring(userID, lhe.StartTime, lhe.EndTime)
		if err != nil {
			return err
		}

		lhe.channelIDs = append(lhe.channelIDs, channelIDs...)
	}

	lhe.channelIDs = deduplicateStringSlice(lhe.channelIDs)

	return nil
}

// ExportData ...
func (lhe *LegalHoldExecution) ExportData() error {
	for _, channelID := range lhe.channelIDs {
		cursor := model.NewLegalHoldCursor(lhe.StartTime)
		for {
			var posts []model.LegalHoldPost
			var err error

			posts, cursor, err = lhe.store.LegalholdExport(channelID, lhe.EndTime, cursor, POST_EXPORT_BATCH_LIMIT)
			if err != nil {
				return err
			}

			if len(posts) == 0 {
				break
			}

			// TODO: Write those lines to the appropriate file-backend file.

			if len(posts) < POST_EXPORT_BATCH_LIMIT {
				break
			}
		}
	}

	return nil
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
