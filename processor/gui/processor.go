package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/cmd"
	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

// processLegalHold executes the legal hold processing with progress updates
func processLegalHold(dataPath, outputPath, secret string, logCallback func(string)) (string, error) {
	// Validate input file is a zip
	if !strings.HasSuffix(strings.ToLower(dataPath), ".zip") {
		return "", fmt.Errorf("legal hold data must be a ZIP file: %s", dataPath)
	}

	// Validate output path is a directory
	if !isDirectory(outputPath) {
		return "", fmt.Errorf("output path must be a directory: %s", outputPath)
	}

	// Create pipe for capturing output
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	// Start goroutine to read from pipe and send to GUI
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			logCallback(scanner.Text() + "\n")
		}
	}()

	// Restore original stdout/stderr when done
	defer func() {
		w.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	opts := model.LegalHoldProcessOptions{
		LegalHoldData:   dataPath,
		OutputPath:      outputPath,
		LegalHoldSecret: secret,
	}

	_, err := cmd.ProcessLegalHolds(opts)
	if err != nil {
		return "", fmt.Errorf("failed to process legal hold: %w", err)
	}

	// Return path to index.html
	indexPath := filepath.Join(outputPath, "index.html")
	return indexPath, nil
}

// isDirectory checks if the given path is a directory
func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
