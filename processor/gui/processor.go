package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/cmd"
	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

// processLegalHold executes the legal hold processing with progress updates
func processLegalHold(dataPath, outputPath, secret string) error {
	// Validate that paths are directories
	if !isDirectory(dataPath) {
		return fmt.Errorf("legal hold path must be a directory: %s", dataPath)
	}

	if !isDirectory(outputPath) {
		return fmt.Errorf("output path must be a directory: %s", outputPath)
	}

	// Create a LegalHold instance from the directory
	hold := model.LegalHold{
		Path: dataPath,
		Name: filepath.Base(dataPath),
		ID:   filepath.Base(dataPath), // Using directory name as ID
	}

	// Process the legal hold using the existing cmd package
	err := cmd.ProcessLegalHold(hold, outputPath)
	if err != nil {
		return fmt.Errorf("failed to process legal hold: %w", err)
	}

	return nil
}

// isDirectory checks if the given path is a directory
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
