package parse

import (
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
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
			channels = append(channels, model.NewChannel(filepath.Join(legalHold.Path, entry.Name()), entry.Name()))
		}
	}

	return channels, nil
}

// ListChannelsFromChannelMemberships takes a ChannelMemberships object from
// the export index and returns a list of model.Channel objects populated from
// their data.
func ListChannelsFromChannelMemberships(memberships []model.LegalHoldChannelMembership, legalHold model.LegalHold) []model.Channel {
	var channels []model.Channel

	for _, membership := range memberships {
		channels = append(
			channels,
			model.NewChannelWithBounds(
				filepath.Join(legalHold.Path, membership.ChannelID),
				membership.ChannelID,
				membership.StartTime,
				membership.EndTime,
			),
		)
	}

	return channels
}
