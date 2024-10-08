package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func ParseHashes(tempPath, lhPath, secret string) error {
	var hashes map[string]string

	fileHandle, err := os.Open(filepath.Join(lhPath, model.HashesPath))
	if err != nil {
		return fmt.Errorf("error opening hashes.json file: %w", err)
	}

	decoder := json.NewDecoder(fileHandle)
	err = decoder.Decode(&hashes)
	if err != nil {
		return fmt.Errorf("error decoding hashes.json file: %w", err)
	}

	for path, hash := range hashes {
		hashReader, err := os.Open(filepath.Join(tempPath, path))
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}

		fileHash, err := model.HashReader(secret, hashReader)
		if err != nil {
			return fmt.Errorf("error reading hash: %w", err)
		}

		if fileHash != hash {
			return fmt.Errorf("hash mismatch for file: %s", path)
		}
	}

	return nil
}
