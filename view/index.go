package view

import (
	"fmt"
	"github.com/grundleborg/mattermost-legal-hold-processor/model"
	"html/template"
	"os"
	"path/filepath"
)

type User struct {
	User  model.User
	Teams []*UserTeam
}

type UserTeam struct {
	TeamData *model.LegalHoldTeam
	Channels []*UserChannel
}

type UserChannel struct {
	ChannelData *model.LegalHoldChannel
}

func WriteIndexFile(legalHold model.LegalHold, legalHoldIndex model.LegalHoldIndex, teamLookup model.TeamLookup, channelLookup model.ChannelLookup, teamForChannelLookup model.TeamForChannelLookup, outputPath string) error {
	data := struct {
		LegalHold *model.LegalHold
		Index     *model.LegalHoldIndex
		Users     []User
	}{
		LegalHold: &legalHold,
		Index:     &legalHoldIndex,
		Users:     []User{},
	}

	for userID, userIndex := range legalHoldIndex.Users {
		user := User{
			User:  model.NewUserFromIDAndIndex(userID, userIndex),
			Teams: []*UserTeam{},
		}

		for _, channelIndex := range userIndex.Channels {
			team := teamForChannelLookup[channelIndex.ChannelID]

			userTeam := &UserTeam{
				TeamData: team,
			}

			found := false
			for _, t := range user.Teams {
				if t.TeamData.ID == team.ID {
					userTeam = t
					found = true
					break
				}
			}

			userTeam.Channels = append(userTeam.Channels, &UserChannel{
				ChannelData: channelLookup[channelIndex.ChannelID],
			})

			if !found {
				user.Teams = append(user.Teams, userTeam)
			}
		}

		data.Users = append(data.Users, user)
	}

	tmpl, err := template.ParseFS(templates, "templates/index.html")
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
