package azureblob

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/stretchr/testify/require"
)

type fakeDownloader struct {
	body io.ReadCloser
	ctx  context.Context
}

func (f *fakeDownloader) DownloadStream(ctx context.Context, _ *blob.DownloadStreamOptions) (blob.DownloadStreamResponse, error) {
	f.ctx = ctx
	return blob.DownloadStreamResponse{DownloadResponse: blob.DownloadResponse{Body: f.body}}, nil
}

func TestRangeReaderStaysOpenPastTheRequestTimeout(t *testing.T) {
	payload := []byte("hold-bytes")
	downloader := &fakeDownloader{body: io.NopCloser(bytes.NewReader(payload))}
	reader := newRangeReader(downloader, int64(len(payload)), time.Hour)
	defer func() { _ = reader.Close() }()

	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, payload, got)
	require.NoError(t, reader.ctx.Err())
	require.NoError(t, downloader.ctx.Err())
}

type stallBody struct {
	ctx context.Context
}

func (b *stallBody) Read([]byte) (int, error) {
	<-b.ctx.Done()
	return 0, b.ctx.Err()
}

func (b *stallBody) Close() error { return nil }

type stallDownloader struct{}

func (s *stallDownloader) DownloadStream(ctx context.Context, _ *blob.DownloadStreamOptions) (blob.DownloadStreamResponse, error) {
	return blob.DownloadStreamResponse{DownloadResponse: blob.DownloadResponse{Body: &stallBody{ctx: ctx}}}, nil
}

func TestRangeReaderCancelsAStalledBody(t *testing.T) {
	reader := newRangeReader(&stallDownloader{}, 10, 20*time.Millisecond)
	defer func() { _ = reader.Close() }()
	_, err := reader.Read(make([]byte, 4))
	require.Error(t, err)
}
