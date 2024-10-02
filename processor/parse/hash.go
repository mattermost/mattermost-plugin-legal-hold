package parse

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

func ParseHashes(tempPath, lhPath, secret string) error {
	var hashes []model.Hash

	fileHandle, err := os.Open(filepath.Join(lhPath, model.HashesPath))
	if err != nil {
		return fmt.Errorf("error opening hashes csv file: %w", err)
	}

	err = gocsv.UnmarshalWithoutHeaders(fileHandle, &hashes)
	if err != nil {
		return fmt.Errorf("error parsing hashes csv file: %w", err)
	}

	for _, hash := range hashes {
		hashReader, err := os.Open(filepath.Join(tempPath, hash.Path))
		if err != nil {
			return fmt.Errorf("error opening file: %w", err)
		}

		fileHash, err := model.HashReader(secret, hashReader)
		if err != nil {
			return fmt.Errorf("error reading hash: %w", err)
		}

		if fileHash != hash.Hash {
			return fmt.Errorf("hash mismatch for file: %s", hash.Path)
		}
	}

	return nil
}
