package view

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

// WriteChannel takes the data for the posts in a channel and writes out the page for that channel.
func WriteChannel(hold model.LegalHold, channel model.Channel, posts []*model.PostWithFiles, teamData *model.LegalHoldTeam, channelData *model.LegalHoldChannel, outputPath string) error {
	data := struct {
		Hold        model.LegalHold
		TeamData    *model.LegalHoldTeam
		ChannelData *model.LegalHoldChannel
		Posts       []*model.PostWithFiles
	}{
		Hold:        hold,
		TeamData:    teamData,
		ChannelData: channelData,
		Posts:       posts,
	}

	tmpl, err := template.ParseFS(templates, "templates/channel.html")
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(outputPath, fmt.Sprintf("%s.html", channel.ID)))
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

// WriteUserChannel takes the data for the posts in a channel during a user's
// presence in that channel and writes out the page for that channel.
func WriteUserChannel(hold model.LegalHold, user model.User, channel model.Channel, posts []*model.PostWithFiles, teamData *model.LegalHoldTeam, channelData *model.LegalHoldChannel, outputPath string) error {
	data := struct {
		Hold        model.LegalHold
		TeamData    *model.LegalHoldTeam
		ChannelData *model.LegalHoldChannel
		Posts       []*model.PostWithFiles
		User        model.User
	}{
		Hold:        hold,
		TeamData:    teamData,
		ChannelData: channelData,
		Posts:       posts,
		User:        user,
	}

	tmpl, err := template.ParseFS(templates, "templates/user_channel.html")
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(outputPath, fmt.Sprintf("%s_%s.html", user.ID, channel.ID)))
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

type ChannelData struct {
	TeamData    *model.LegalHoldTeam
	ChannelData *model.LegalHoldChannel
	Posts       []*model.PostWithFiles
}

// WriteUserAllChannels writes all data for all channels for a user in one go.
func WriteUserAllChannels(hold model.LegalHold, user model.User, allPosts map[string][]*model.PostWithFiles, teamForChannelLookup model.TeamForChannelLookup, channelLookup model.ChannelLookup, outputPath string) error {
	data := struct {
		Hold     model.LegalHold
		User     model.User
		Channels []ChannelData
	}{
		Hold: hold,
		User: user,
	}

	for channelID, posts := range allPosts {
		data.Channels = append(data.Channels, ChannelData{
			TeamData:    teamForChannelLookup[channelID],
			ChannelData: channelLookup[channelID],
			Posts:       posts,
		})
	}

	tmpl, err := template.ParseFS(templates, "templates/user.html")
	if err != nil {
		return err
	}

	file, err := os.Create(filepath.Join(outputPath, fmt.Sprintf("%s.html", user.ID)))
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
