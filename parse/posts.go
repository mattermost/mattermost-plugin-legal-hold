package parse

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/gocarina/gocsv"

	"github.com/grundleborg/mattermost-legal-hold-processor/model"
)

// LoadPosts creates a list of all posts in the provided channel within the given timestamp range.
func LoadPosts(channel model.Channel) ([]*model.Post, error) {
	var posts []*model.Post
	messagesPath := filepath.Join(channel.Path, "messages")

	fmt.Printf("Reading posts in channel: %s\n", channel.ID)
	fmt.Println()

	// Get all files in the messages directory
	files, err := os.ReadDir(messagesPath)
	if err != nil {
		log.Fatal(err)
	}

	// Remove any directories from the file list
	var onlyFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() {
			onlyFiles = append(onlyFiles, file)
		}
	}

	// Sort the files alphabetically
	sort.Slice(onlyFiles, func(i, j int) bool {
		return onlyFiles[i].Name() < onlyFiles[j].Name()
	})

	// Parse each file
	for _, file := range onlyFiles {

		// Open the file
		fileHandle, err := os.Open(filepath.Join(messagesPath, file.Name()))
		if err != nil {
			return nil, err
		}

		var newPosts []*model.Post

		// Parse the file into posts
		err = gocsv.UnmarshalFile(fileHandle, &newPosts)
		if err != nil {
			return nil, err
		}

		// Close the file
		err = fileHandle.Close()
		if err != nil {
			return nil, err
		}

		// Filter newPosts based on CreateAt timestamp
		for _, post := range newPosts {
			// Checking if the post's CreateAt is within the range of lowerBound and upperBound.
			if post.PostCreateAt >= channel.LowerBound && post.PostCreateAt <= channel.UpperBound {
				posts = append(posts, post)
			}
		}
	}

	return posts, nil
}
