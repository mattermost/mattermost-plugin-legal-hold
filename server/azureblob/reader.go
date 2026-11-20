package azureblob

import (
	"context"
	"io"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/pkg/errors"
)

// blobDownloader is the subset of *blob.Client the range reader uses.
type blobDownloader interface {
	DownloadStream(ctx context.Context, opts *blob.DownloadStreamOptions) (blob.DownloadStreamResponse, error)
}

// rangeReader is a seekable reader over one blob, backed by HTTP Range
// requests. The first Read at the current offset opens a stream. Seek closes
// that stream so the next Read starts at the new offset. Close cancels the
// context. There is no deadline on that context: cancelling it would end a
// download the caller is still reading.
type rangeReader struct {
	ctx        context.Context
	cancel     context.CancelFunc
	blobClient blobDownloader
	size       int64
	offset     int64
	body       io.ReadCloser
}

func newRangeReader(client blobDownloader, size int64) *rangeReader {
	ctx, cancel := context.WithCancel(context.Background())
	return &rangeReader{
		ctx:        ctx,
		cancel:     cancel,
		blobClient: client,
		size:       size,
	}
}

func (r *rangeReader) Read(p []byte) (int, error) {
	if r.offset >= r.size {
		return 0, io.EOF
	}
	if r.body == nil {
		resp, err := r.blobClient.DownloadStream(r.ctx, &blob.DownloadStreamOptions{
			Range: blob.HTTPRange{Offset: r.offset, Count: 0},
		})
		if err != nil {
			return 0, errors.Wrap(err, "failed to open azure range stream")
		}
		r.body = resp.Body
	}
	n, err := r.body.Read(p)
	r.offset += int64(n)
	if err == nil {
		return n, nil
	}
	_ = r.body.Close()
	r.body = nil
	if err == io.EOF && r.offset < r.size {
		return n, io.ErrUnexpectedEOF
	}
	return n, err
}

func (r *rangeReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.offset + offset
	case io.SeekEnd:
		abs = r.size + offset
	default:
		return 0, errors.Errorf("invalid whence: %d", whence)
	}
	if abs < 0 {
		return 0, errors.Errorf("negative position: %d", abs)
	}
	if abs == r.offset {
		return abs, nil
	}
	if r.body != nil {
		_ = r.body.Close()
		r.body = nil
	}
	r.offset = abs
	return abs, nil
}

func (r *rangeReader) Close() error {
	r.cancel()
	if r.body != nil {
		return r.body.Close()
	}
	return nil
}
