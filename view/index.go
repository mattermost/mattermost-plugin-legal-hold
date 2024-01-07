package view

import (
	"fmt"
	"github.com/grundleborg/mattermost-legal-hold-processor/model"
	"html/template"
	"os"
	"path/filepath"
	"slices"
)

func WriteIndexFile(legalHold model.LegalHold, legalHoldIndex model.LegalHoldIndex, outputPath string) error {
	data := struct {
		LegalHold *model.LegalHold
		Index     *model.LegalHoldIndex
		Channels  []string
		Users     []model.UserWithChannels
	}{
		LegalHold: &legalHold,
		Index:     &legalHoldIndex,
		Channels:  []string{},
		Users:     []model.UserWithChannels{},
	}

	for userID, userIndex := range legalHoldIndex {
		user := model.NewUserWithChannelsFromIDAndIndex(userID, userIndex)

		for _, channelIndex := range userIndex.Channels {
			if !slices.Contains(data.Channels, channelIndex.ChannelID) {
				data.Channels = append(data.Channels, channelIndex.ChannelID)
			}
			user.Channels = append(user.Channels, channelIndex.ChannelID)
		}

		data.Users = append(data.Users, user)
	}

	tmpl, err := template.ParseFiles("view/templates/index.html")
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(outputPath, "index.html"))
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Printf("%v\n", err)
		}
	}()

	return tmpl.Execute(file, data)
}
