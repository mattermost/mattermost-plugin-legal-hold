package azureblob

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/stretchr/testify/require"
)

type raceStore struct {
	mu         sync.Mutex
	reads      int
	staged     []byte
	committed  []string
	commitEtag azcore.ETag
}

func (s *raceStore) blockList(context.Context) ([]string, azcore.ETag, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	if s.reads == 1 {
		return []string{"x"}, "etag-1", 1, nil
	}
	return []string{"x", "other"}, "etag-2", 2, nil
}

func (s *raceStore) stage(_ context.Context, r io.Reader, _ int) ([]string, int64, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, err
	}
	s.mu.Lock()
	s.staged = append([]byte{}, buf...)
	s.mu.Unlock()
	return []string{"mine"}, int64(len(buf)), nil
}

func (s *raceStore) open(context.Context, azcore.ETag) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (s *raceStore) commit(_ context.Context, ids []string, etag azcore.ETag) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if etag == "etag-1" {
		return errAppendConflict
	}
	s.committed = append([]string{}, ids...)
	s.commitEtag = etag
	return nil
}

func TestAppendRetriesWhenCommitLosesTheRace(t *testing.T) {
	store := &raceStore{}
	n, err := commitAppend(context.Background(), store, bytes.NewReader([]byte("Z")), "legalhold/index.json")
	require.NoError(t, err)
	require.Equal(t, int64(1), n)
	require.Equal(t, []string{"x", "other", "mine"}, store.committed)
	require.Equal(t, azcore.ETag("etag-2"), store.commitEtag)
	require.Equal(t, []byte("Z"), store.staged)
}

type consolidateStore struct {
	data      []byte
	pieces    map[string][]byte
	next      int
	committed []byte
}

func (s *consolidateStore) blockList(context.Context) ([]string, azcore.ETag, int64, error) {
	ids := make([]string, azureMaxCommittedBlocks-1)
	for i := range ids {
		ids[i] = "old"
	}
	return ids, "etag", int64(len(s.data)), nil
}

func (s *consolidateStore) stage(_ context.Context, r io.Reader, blockSize int) ([]string, int64, error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, 0, err
	}
	if s.pieces == nil {
		s.pieces = map[string][]byte{}
	}
	if len(buf) == 0 {
		return nil, 0, nil
	}
	var ids []string
	for start := 0; start < len(buf); start += blockSize {
		end := start + blockSize
		if end > len(buf) {
			end = len(buf)
		}
		id := fmt.Sprintf("s%d", s.next)
		s.next++
		s.pieces[id] = append([]byte{}, buf[start:end]...)
		ids = append(ids, id)
	}
	return ids, int64(len(buf)), nil
}

func (s *consolidateStore) open(context.Context, azcore.ETag) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.data)), nil
}

func (s *consolidateStore) commit(_ context.Context, ids []string, _ azcore.ETag) error {
	if len(ids) >= azureMaxCommittedBlocks {
		return errAppendConflict
	}
	var got []byte
	for _, id := range ids {
		piece, ok := s.pieces[id]
		if !ok {
			return errAppendConflict
		}
		got = append(got, piece...)
	}
	s.committed = got
	return nil
}

func TestAppendConsolidatesBeforeTheBlockCap(t *testing.T) {
	original := bytes.Repeat([]byte("a"), 32)
	store := &consolidateStore{data: original}
	n, err := commitAppend(context.Background(), store, bytes.NewReader([]byte("ZZ")), "legalhold/data.bin")
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
	require.Equal(t, append(append([]byte{}, original...), 'Z', 'Z'), store.committed)
	require.Less(t, len(store.pieces), azureMaxCommittedBlocks)
}

func TestConsolidateBlockSizeRejectsAnImpossibleBlob(t *testing.T) {
	_, err := consolidateBlockSize(azureConsolidateBlockBytes*int64(azureMaxCommittedBlocks), 1)
	require.Error(t, err)
}
