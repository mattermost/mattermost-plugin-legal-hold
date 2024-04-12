package parse

import (
	"errors"
	"maps"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func ProcessFiles(legalHold model.LegalHold) (model.FileLookup, error) {
	dirEntries, err := os.ReadDir(legalHold.Path)
	if err != nil {
		return nil, err
	}

	fileLookup := make(model.FileLookup)
	for _, entry := range dirEntries {
		if entry.IsDir() {
			extra, err := processFilesInChannel(filepath.Join(legalHold.Path, entry.Name()))
			if err != nil {
				return nil, err
			}

			maps.Copy(fileLookup, extra)
		}
	}

	return fileLookup, nil
}

func processFilesInChannel(path string) (model.FileLookup, error) {
	// Check if the "files" directory exists. Continue to next channel if not.
	if _, err := os.Stat(filepath.Join(path, "files")); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	fileLookup := make(model.FileLookup)

	// Loop through nested sub-folders to find files.
	err := filepath.WalkDir(filepath.Join(path, "files"), func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Immediate parent directory name is the FileID
		if !d.IsDir() {
			fileID := filepath.Base(filepath.Dir(filePath))
			fileLookup[fileID] = filepath.Join(filePath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return fileLookup, nil
}
