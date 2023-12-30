package parse

import (
	"os"
	"path/filepath"

	"github.com/grundleborg/mattermost-legal-hold-processor/model"
)

// ListChannels retrieves a list of model.Channel objects from the specified LegalHold.
func ListChannels(legalHold model.LegalHold) ([]model.Channel, error) {
	var channels []model.Channel
	dirEntries, err := os.ReadDir(legalHold.Path)
	if err != nil {
		return nil, err
	}

	for _, entry := range dirEntries {
		if entry.IsDir() {
			channels = append(channels, model.Channel{ID: entry.Name(), Path: filepath.Join(legalHold.Path, entry.Name())})
		}
	}

	return channels, nil
}
