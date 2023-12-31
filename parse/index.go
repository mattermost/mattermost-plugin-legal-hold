package parse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grundleborg/mattermost-legal-hold-processor/model"
)

func LoadIndex(legalHold model.LegalHold) (model.LegalHoldIndex, error) {
	filePath := filepath.Join(legalHold.Path, "index.json")

	file, err := os.Open(filePath)
	if err != nil {
		return model.LegalHoldIndex{}, err
	}
	defer func() {
		if err = file.Close(); err != nil {
			fmt.Println(err.Error())
		}
	}()

	index := model.LegalHoldIndex{}
	err = json.NewDecoder(file).Decode(&index)
	if err != nil {
		return model.LegalHoldIndex{}, err
	}

	return index, nil
}
