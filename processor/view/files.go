package view

import (
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func MoveFiles(originalFileLookup model.FileLookup, outputPath string) (model.FileLookup, error) {
	fileLookup := make(model.FileLookup)

	for id, path := range originalFileLookup {
		// Check if input file path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File has already been moved.
			continue
		} else if err != nil {
			return nil, err
		}

		outputDirectory := filepath.Join(outputPath, "files", id)

		// Create outputDirectory if it doesn't exist
		if _, err := os.Stat(outputDirectory); os.IsNotExist(err) {
			err := os.MkdirAll(outputDirectory, os.ModePerm)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		}

		// Move the file
		destination := filepath.Join(outputDirectory, filepath.Base(path))
		err := os.Rename(path, destination)
		if err != nil {
			return nil, err
		}

		// Add it's new path to the new lookup
		fileLookup[id] = destination
	}

	return originalFileLookup, nil
}
