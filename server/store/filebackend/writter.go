package filebackend

import (
	"bytes"
	"io"
	"log"

	"github.com/mattermost/mattermost/server/v8/platform/shared/filestore"
)

// fileBackendWritter is a simple io.Writer that writes to a file using a filestore.FileBackend
type fileBackendWritter struct {
	filePath    string
	fileBackend filestore.FileBackend
	// created is used to know if the file has been created or not, to use either WriteFile or AppendFile
	created bool
}

func (s *fileBackendWritter) Write(p []byte) (n int, err error) {
	var written int64
	if !s.created {
		s.created = true
		log.Println("writeFile")
		written, err = s.fileBackend.WriteFile(bytes.NewReader(p), s.filePath)
	} else {
		log.Println("appendFile")
		written, err = s.fileBackend.AppendFile(bytes.NewReader(p), s.filePath)
	}
	return int(written), err
}

func NewFileBackendWritter(fileBackend filestore.FileBackend, filePath string) io.Writer {
	return &fileBackendWritter{
		filePath:    filePath,
		fileBackend: fileBackend,
	}
}
