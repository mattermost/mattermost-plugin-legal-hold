package view

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
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
			// Get team data from lookup, or create fallback for DMs/GMs
			team := teamForChannelLookup[channelIndex.ChannelID]
			if team == nil {
				// Fallback for DMs/GMs which don't belong to a team
				team = &model.LegalHoldTeam{
					ID:          "",
					Name:        "Direct Messages",
					DisplayName: "Direct Messages",
				}
			}

			// Get channel data from lookup, or create fallback
			channel := channelLookup[channelIndex.ChannelID]
			if channel == nil {
				channel = &model.LegalHoldChannel{
					ID:          channelIndex.ChannelID,
					Name:        "Direct/Group Message",
					DisplayName: "Direct/Group Message",
					Type:        "D",
				}
			}

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
				ChannelData: channel,
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

	path := filepath.Join(outputPath, "index.html")
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Printf("%v\n", err)
		}
	}()

	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	fmt.Printf("Browse the HTML output at: %s\n\n", path)

	return tmpl.Execute(file, data)
}
