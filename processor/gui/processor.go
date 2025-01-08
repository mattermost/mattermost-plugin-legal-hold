package main

import (
	"fmt"
	"path/filepath"
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

	// TODO: We need to modify the existing cmd.Process function to:
	// 1. Accept these parameters directly instead of using cobra
	// 2. Provide progress updates that can be shown in the GUI
	// 3. Handle errors appropriately for GUI display

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
