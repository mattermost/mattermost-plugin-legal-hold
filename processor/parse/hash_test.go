package parse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHashes(t *testing.T) {
	testCases := []struct {
		name        string
		hashesFile  string
		fileJSON    string
		secret      string
		expectedErr bool
	}{
		{
			name:       "correct hash",
			hashesFile: `{"file.json": "59a8ed5870c634232d8ec53ab3fc0521e448894950f25dccbf7a4e6a0b4c8b34b59585a56ac86eabef8e8c246c99197285ec644db9798fad062088a45acd1afb"}`,
			fileJSON:   `{"key": "value"}`,
			secret:     "1234",
		},
		{
			name:        "incorrect secret",
			hashesFile:  `{"file.json": "59a8ed5870c634232d8ec53ab3fc0521e448894950f25dccbf7a4e6a0b4c8b34b59585a56ac86eabef8e8c246c99197285ec644db9798fad062088a45acd1afb"}`,
			fileJSON:    `{"key": "value"}`,
			secret:      "nometokens",
			expectedErr: true,
		},
		{
			name:        "incorrect file contents",
			hashesFile:  `{"file.json": "59a8ed5870c634232d8ec53ab3fc0521e448894950f25dccbf7a4e6a0b4c8b34b59585a56ac86eabef8e8c246c99197285ec644db9798fad062088a45acd1afb"}`,
			fileJSON:    `{bogus}`,
			secret:      "1234",
			expectedErr: true,
		},
		{
			name:        "incorrect hash",
			hashesFile:  `{"file.json": "xxx"}`,
			fileJSON:    `{"key": "value"}`,
			secret:      "1234",
			expectedErr: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "")
			require.NoError(t, err)

			legalHoldPath := filepath.Join(tempDir, "/legalhold")
			os.MkdirAll(legalHoldPath, 0755)

			err = os.WriteFile(tempDir+"/hashes.json", []byte(testCase.hashesFile), 0644)
			require.NoError(t, err)

			err = os.WriteFile(tempDir+"/file.json", []byte(testCase.fileJSON), 0644)
			require.NoError(t, err)

			err = ParseHashes(tempDir, legalHoldPath, testCase.secret)
			if testCase.expectedErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
