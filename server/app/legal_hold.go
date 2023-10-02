package app

import "github.com/mattermost/mattermost-plugin-legal-hold/server/store"

type LegalHoldExecution struct {
	StartTime int64
	EndTime   int64
	UserIDs   []string

	store *store.SQLStore

	channelIDs []string
}

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
