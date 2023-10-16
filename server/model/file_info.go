package model

// FileInfo is an abridged version of the FileInfo struct from
// the main Mattermost model.
type FileInfo struct {
	Id       string
	Path     string
	Name     string
	Size     int64
	MimeType string
}
