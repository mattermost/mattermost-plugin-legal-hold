package view

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func MoveFiles(originalFileLookup model.FileLookup, outputPath string) (model.FileLookup, error) {
	fileLookup := make(model.FileLookup)

	for id, path := range originalFileLookup {
		fmt.Printf("Moving file %s at %s\n", id, path)
		// Check if input file path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			// File has already been moved.
			continue
		} else if err != nil {
			return nil, err
		}

		outputDirectory := filepath.Join(outputPath, "files", id)
		fmt.Printf("output dir: %s\n", outputDirectory)

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
		fileLookup[id] = filepath.Join("files", id, filepath.Base(path))
		fmt.Printf("Destination: %s\n", destination)
	}

	return fileLookup, nil
}
