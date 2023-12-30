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
		Hold    model.LegalHold
		Channel model.Channel
		Posts   []*model.Post
	}{
		Hold:    hold,
		Channel: channel,
		Posts:   posts,
	}

	tmpl, err := template.ParseFiles("view/templates/channel.html")
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
