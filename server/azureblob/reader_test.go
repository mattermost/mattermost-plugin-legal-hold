package azureblob

import (
	"bytes"
	"context"
	"io"
	"testing"

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
	reader := newRangeReader(downloader, int64(len(payload)))
	defer reader.Close()

	got, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, payload, got)
	require.NoError(t, reader.ctx.Err())
	require.NoError(t, downloader.ctx.Err())
}
