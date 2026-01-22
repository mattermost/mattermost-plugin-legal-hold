package view

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

// GetChannelAndTeamData retrieves channel and team data from lookups, or creates fallback data
// for channels not in the index (e.g., Direct Messages or Group Messages that don't belong to teams).
func GetChannelAndTeamData(channelID string, firstPost *model.Post, channelLookup model.ChannelLookup, teamForChannelLookup model.TeamForChannelLookup) (*model.LegalHoldChannel, *model.LegalHoldTeam) {
	// Try to get channel data from lookup
	channelData := channelLookup[channelID]
	if channelData == nil && firstPost != nil {
		// Use data from the first post for channels not in the index (DMs/GMs)
		channelData = &model.LegalHoldChannel{
			ID:          channelID,
			Name:        firstPost.ChannelName,
			DisplayName: firstPost.ChannelDisplayName,
			Type:        firstPost.ChannelType,
		}
	} else if channelData == nil {
		// Fallback if no posts exist - must be a DM/GM since it's not in a team
		channelData = &model.LegalHoldChannel{
			ID:          channelID,
			Name:        "Direct/Group Message",
			DisplayName: "Direct/Group Message",
			Type:        "D",
		}
	}

	// Try to get team data from lookup
	teamData := teamForChannelLookup[channelID]
	if teamData == nil && firstPost != nil {
		// Use data from the first post for channels not in a team (DMs/GMs)
		teamData = &model.LegalHoldTeam{
			ID:          firstPost.TeamID,
			Name:        firstPost.TeamName,
			DisplayName: firstPost.TeamDisplayName,
		}
	} else if teamData == nil {
		// Fallback for DMs/GMs which don't belong to a team
		teamData = &model.LegalHoldTeam{
			ID:          "",
			Name:        "No Team",
			DisplayName: "No Team",
		}
	}

	return channelData, teamData
}

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
		// Get channel and team data from lookups, or create fallback if not found
		var firstPost *model.Post
		if len(posts) > 0 {
			firstPost = posts[0].Post
		}
		channelData, teamData := GetChannelAndTeamData(channelID, firstPost, channelLookup, teamForChannelLookup)

		data.Channels = append(data.Channels, ChannelData{
			TeamData:    teamData,
			ChannelData: channelData,
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
