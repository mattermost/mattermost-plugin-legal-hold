package legalhold

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/mattermost/mattermost-server/v6/shared/filestore"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-legal-hold/server/model"
)

func TestCopyAttachmentStreamsAndHashes(t *testing.T) {
	payload := []byte("hold-bytes")
	source := newMemBackend(map[string][]byte{"src/a.txt": payload})
	dest := newMemBackend(nil)
	ex := Execution{
		LegalHold:         &model.LegalHold{Secret: "hold-secret"},
		fileBackend:       dest,
		sourceFileBackend: source,
		hashes:            model.HashList{},
	}

	require.NoError(t, ex.copyAttachment("src/a.txt", "dest/a.txt"))
	require.Equal(t, payload, dest.files["dest/a.txt"])

	hasher := hmac.New(sha512.New, []byte("hold-secret"))
	_, _ = hasher.Write(payload)
	require.Equal(t, fmt.Sprintf("%x", hasher.Sum(nil)), ex.hashes["dest/a.txt"])
}

func TestCopyAttachmentMissingFileIsNotARetryableError(t *testing.T) {
	ex := Execution{
		LegalHold:         &model.LegalHold{Secret: "hold-secret"},
		fileBackend:       newMemBackend(nil),
		sourceFileBackend: newMemBackend(nil),
		hashes:            model.HashList{},
	}

	err := ex.copyAttachment("missing", "dest/missing")
	require.ErrorIs(t, err, errAttachmentNotFound)
}

func TestCopyAttachmentRejectsAShortWrite(t *testing.T) {
	payload := []byte("hold-bytes")
	source := newMemBackend(map[string][]byte{"src/a.txt": payload})
	dest := shortWriteBackend{memBackend: newMemBackend(nil)}
	ex := Execution{
		LegalHold:         &model.LegalHold{Secret: "hold-secret"},
		fileBackend:       dest,
		sourceFileBackend: source,
		hashes:            model.HashList{},
	}

	err := ex.copyAttachment("src/a.txt", "dest/a.txt")
	require.Error(t, err)
	require.NotErrorIs(t, err, errAttachmentNotFound)
	require.Empty(t, ex.hashes)
}

func TestCopyAttachmentDoesNotSkipAMissingDestination(t *testing.T) {
	source := newMemBackend(map[string][]byte{"src/a.txt": []byte("hold-bytes")})
	dest := missingDestBackend{memBackend: newMemBackend(nil)}
	ex := Execution{
		LegalHold:         &model.LegalHold{Secret: "hold-secret"},
		fileBackend:       dest,
		sourceFileBackend: source,
		hashes:            model.HashList{},
	}

	err := ex.copyAttachment("src/a.txt", "dest/a.txt")
	require.Error(t, err)
	require.NotErrorIs(t, err, errAttachmentNotFound)
	require.Empty(t, ex.hashes)
}

func TestIsSourceMissingIgnoresOtherFailures(t *testing.T) {
	require.True(t, isSourceMissing(fmt.Errorf("open: %w", os.ErrNotExist)))
	require.False(t, isSourceMissing(errors.New("open: no such file or directory")))
	require.False(t, isSourceMissing(fmt.Errorf("unable to open file NoSuchKey.txt: %w", os.ErrPermission)))
	require.False(t, isSourceMissing(errors.New("The specified key does not exist.")))
	require.True(t, isSourceMissing(fmt.Errorf("get object: %w", minio.ErrorResponse{Code: "NoSuchKey"})))
	require.False(t, isSourceMissing(fmt.Errorf("get object: %w", minio.ErrorResponse{Code: "AccessDenied"})))
	require.False(t, isSourceMissing(errors.New("connection reset")))
}

// shortWriteBackend reports success after reading the body, but returns a
// count shorter than the source. That is the Azure stageBlocks result when a
// download ends early.
type shortWriteBackend struct {
	*memBackend
}

func (s shortWriteBackend) WriteFile(r io.Reader, path string) (int64, error) {
	if _, err := io.Copy(io.Discard, r); err != nil {
		return 0, err
	}
	return 1, nil
}

type missingDestBackend struct {
	*memBackend
}

func (m missingDestBackend) WriteFile(io.Reader, string) (int64, error) {
	return 0, fmt.Errorf("open dest: %w", os.ErrNotExist)
}

type memBackend struct {
	files map[string][]byte
}

func newMemBackend(files map[string][]byte) *memBackend {
	if files == nil {
		files = map[string][]byte{}
	}
	return &memBackend{files: files}
}

func (m *memBackend) TestConnection() error { return nil }

func (m *memBackend) Reader(path string) (filestore.ReadCloseSeeker, error) {
	payload, ok := m.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return nopReadSeeker{Reader: bytes.NewReader(payload)}, nil
}

func (m *memBackend) ReadFile(path string) ([]byte, error) {
	payload, ok := m.files[path]
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return append([]byte(nil), payload...), nil
}

func (m *memBackend) FileExists(path string) (bool, error) {
	_, ok := m.files[path]
	return ok, nil
}

func (m *memBackend) FileSize(path string) (int64, error) {
	payload, ok := m.files[path]
	if !ok {
		return 0, io.ErrUnexpectedEOF
	}
	return int64(len(payload)), nil
}

func (m *memBackend) CopyFile(oldPath, newPath string) error {
	payload, ok := m.files[oldPath]
	if !ok {
		return io.ErrUnexpectedEOF
	}
	m.files[newPath] = append([]byte(nil), payload...)
	return nil
}

func (m *memBackend) MoveFile(oldPath, newPath string) error {
	if err := m.CopyFile(oldPath, newPath); err != nil {
		return err
	}
	delete(m.files, oldPath)
	return nil
}

func (m *memBackend) WriteFile(fr io.Reader, path string) (int64, error) {
	payload, err := io.ReadAll(fr)
	if err != nil {
		return 0, err
	}
	m.files[path] = payload
	return int64(len(payload)), nil
}

func (m *memBackend) AppendFile(io.Reader, string) (int64, error) {
	return 0, errors.New("not implemented")
}

func (m *memBackend) RemoveFile(path string) error {
	delete(m.files, path)
	return nil
}

func (m *memBackend) FileModTime(string) (time.Time, error) { return time.Time{}, nil }

func (m *memBackend) ListDirectory(string) ([]string, error) { return nil, nil }

func (m *memBackend) ListDirectoryRecursively(string) ([]string, error) { return nil, nil }

func (m *memBackend) RemoveDirectory(string) error { return nil }

type nopReadSeeker struct{ *bytes.Reader }

func (nopReadSeeker) Close() error { return nil }
