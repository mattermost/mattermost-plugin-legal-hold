package view

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/grundleborg/mattermost-legal-hold-processor/model"
)

// WriteChannel takes the data for the posts in a channel and writes out the page for that channel.
func WriteChannel(hold model.LegalHold, channel model.Channel, posts []*model.Post, outputPath string) error {
	data := struct {
		Hold               model.LegalHold
		Channel            model.Channel
		ChannelName        string
		ChannelDisplayName string
		TeamName           string
		TeamDisplayName    string
		Posts              []*model.Post
	}{
		Hold:    hold,
		Channel: channel,
		Posts:   posts,
	}

	if len(posts) > 0 {
		data.ChannelName = posts[0].ChannelName
		data.ChannelDisplayName = posts[0].ChannelDisplayName
		data.TeamName = posts[0].TeamName
		data.TeamDisplayName = posts[0].TeamDisplayName
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
func WriteUserChannel(hold model.LegalHold, user model.User, channel model.Channel, posts []*model.Post, outputPath string) error {
	data := struct {
		Hold               model.LegalHold
		Channel            model.Channel
		ChannelName        string
		ChannelDisplayName string
		TeamName           string
		TeamDisplayName    string
		Posts              []*model.Post
		User               model.User
	}{
		Hold:    hold,
		Channel: channel,
		Posts:   posts,
		User:    user,
	}

	if len(posts) > 0 {
		data.ChannelName = posts[0].ChannelName
		data.ChannelDisplayName = posts[0].ChannelDisplayName
		data.TeamName = posts[0].TeamName
		data.TeamDisplayName = posts[0].TeamDisplayName
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
