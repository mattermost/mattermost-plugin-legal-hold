package parse

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattermost/mattermost-plugin-legal-hold/processor/model"
)

// ListLegalHolds retrieves a list of LegalHold objects from the specified directory path
// containing an unpacked legal hold export.
func ListLegalHolds(tempPath string) ([]model.LegalHold, error) {
	legalHoldsPath := filepath.Join(tempPath, "legal_hold")

	files, err := os.ReadDir(legalHoldsPath)
	if err != nil {
		return nil, err
	}

	var legalHolds []model.LegalHold
	for _, file := range files {
		if !file.IsDir() {
			continue
		}

		nameID := strings.Split(file.Name(), "_")
		if len(nameID) != 2 {
			return nil, errors.New("directory name does not match pattern name_id")
		}

		id := strings.TrimSuffix(nameID[1], ")")
		legalHolds = append(legalHolds, model.LegalHold{Path: filepath.Join(legalHoldsPath, file.Name()), Name: nameID[0], ID: id})
	}

	return legalHolds, nil
}
