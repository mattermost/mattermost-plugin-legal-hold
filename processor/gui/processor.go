package main

import (
	"fmt"
	"os"

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

	opts := model.LegalHoldProcessOptions{
		LegalHoldData:   dataPath,
		OutputPath:      outputPath,
		LegalHoldSecret: secret,
	}

	result, err := cmd.ProcessLegalHolds(opts)
	if err != nil {
		return fmt.Errorf("failed to process legal hold: %w", err)
	}

	fmt.Printf("Processed %d legal holds and %d files\n", len(result.LegalHolds), result.FilesCount)
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
