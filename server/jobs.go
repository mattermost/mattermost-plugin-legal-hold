package main

import (
	"path/filepath"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

// JobCleanupOldBundlesFromFilestore is a job that cleans old legal hold bundles from the filestore by
// checking the timestamp in the filename and ensuring that bundles older than 24h are deleted.
func (p *Plugin) jobCleanupOldBundlesFromFilestore() {
	p.API.LogDebug("Starting legal hold cleanup job")

	files, jobErr := p.FileBackend.ListDirectory(model.FilestoreBundlePath)
	if jobErr != nil {
		p.Client.Log.Error("failed to list directory", "err", jobErr)
		return
	}

	for _, file := range files {
		parts := model.FilestoreBundleRegex.FindStringSubmatch(filepath.Base(file))
		if len(parts) != 3 {
			p.Client.Log.Error("Skipping file", "file", file, "reason", "does not match regex", "parts", parts)
			continue
		}

		// golang parse unix time
		parsedTimestamp, errStrConv := strconv.ParseInt(parts[2], 10, 64)
		if errStrConv != nil {
			p.Client.Log.Error("Skipping file", "file", file, "reason", "failed to parse timestamp", "err", errStrConv)
			continue
		}
		fileCreationTime := time.Unix(parsedTimestamp, 0)
		if time.Since(fileCreationTime) > time.Hour*24 {
			p.Client.Log.Debug("Deleting file", "file", file)
			if err := p.FileBackend.RemoveFile(file); err != nil {
				p.Client.Log.Error("Failed to delete file", "file", file, "err", err)
			}
		}

		p.Client.Log.Debug("Checking file", "file", file, "parts", parts)
	}

	p.API.LogDebug("Finished legal hold cleanup job")
}
