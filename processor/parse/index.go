package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func LoadIndex(legalHold model.LegalHold) (model.LegalHoldIndex, error) {
	filePath := filepath.Join(legalHold.Path, "index.json")

	file, err := os.Open(filePath)
	if err != nil {
		return model.LegalHoldIndex{}, err
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Println(err.Error())
		}
	}()

	index := model.LegalHoldIndex{}
	err = json.NewDecoder(file).Decode(&index)
	if err != nil {
		return model.LegalHoldIndex{}, err
	}

	return index, nil
}

func CreateTeamAndChannelLookup(index model.LegalHoldIndex) (model.TeamLookup, model.ChannelLookup, model.TeamForChannelLookup) {
	teamLookup := make(model.TeamLookup)
	channelLookup := make(model.ChannelLookup)
	teamForChannelLookup := make(model.TeamForChannelLookup)
	for _, team := range index.Teams {
		teamLookup[team.ID] = team
		for _, channel := range team.Channels {
			channelLookup[channel.ID] = channel
			teamForChannelLookup[channel.ID] = team
		}
	}

	return teamLookup, channelLookup, teamForChannelLookup
}
