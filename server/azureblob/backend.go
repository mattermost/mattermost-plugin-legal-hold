package azureblob

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/bloberror"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blockblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/container"
	"github.com/google/uuid"
	"github.com/mattermost/mattermost-server/v6/shared/filestore"
)

const (
	// azureBlockSize matches the Azure SDK UploadStream default and stays under
	// the per-block REST limit. Every write goes through StageBlock so a later
	// append can commit a new block list without clobbering a Put Blob.
	azureBlockSize = 4 * 1024 * 1024

	// Azure rejects a Commit Block List of 50,000 or more blocks.
	azureMaxCommittedBlocks = 50000

	// Consolidating a blob near that limit streams it back in chunks no larger
	// than this, so one append cannot pin multiple gigabytes in the plugin.
	azureConsolidateBlockBytes = 64 * 1024 * 1024

	appendCommitAttempts = 8
)

// errAppendConflict is the local stand-in for Azure's 412 ConditionNotMet.
// Commit retries while this, or the service error, is what came back.
var errAppendConflict = errors.New("azure append commit conflict")

// Backend stores legal-hold objects in Azure Blob Storage.
// Two auth modes match the server: shared key, and DefaultAzureCredential
// (managed identity, workload identity, or service principal env vars).
type Backend struct {
	client     *azblob.Client
	container  string
	pathPrefix string
	timeout    time.Duration
}

var _ filestore.FileBackend = (*Backend)(nil)

func New(settings Settings) (*Backend, error) {
	settings = settings.withDefaults()
	// prefix() cleans the joined key, and stripPrefix removes this stored value.
	// Cleaning once keeps "./holds" from being written as "holds/..." and then
	// listed back with the "./holds" prefix still attached.
	settings.PathPrefix = cleanPathPrefix(settings.PathPrefix)
	if err := settings.CheckMandatory(); err != nil {
		return nil, err
	}
	serviceURL, err := ServiceURL(settings)
	if err != nil {
		return nil, err
	}
	client, err := newClient(settings, serviceURL)
	if err != nil {
		return nil, err
	}
	return &Backend{
		client:     client,
		container:  settings.Container,
		pathPrefix: settings.PathPrefix,
		timeout:    time.Duration(settings.timeoutMilliseconds()) * time.Millisecond,
	}, nil
}

func newClient(settings Settings, serviceURL string) (*azblob.Client, error) {
	options := clientOptions(settings)
	switch settings.AuthMode {
	case AuthModeDefaultCredential:
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure default credential: %w", err)
		}
		client, err := azblob.NewClient(serviceURL, cred, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure blob client: %w", err)
		}
		return client, nil
	case AuthModeSharedKey:
		cred, err := azblob.NewSharedKeyCredential(settings.StorageAccount, settings.AccessKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure shared key credential: %w", err)
		}
		client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, cred, options)
		if err != nil {
			return nil, fmt.Errorf("failed to create azure blob client: %w", err)
		}
		return client, nil
	default:
		return nil, fmt.Errorf("unknown azure auth mode %q", settings.AuthMode)
	}
}

// clientOptions uses the SDK default transport for the public Azure clouds.
// A private endpoint still presents the commercial or government hostname, so
// the default transport is the right one there. Custom clouds (Azurite, a
// proxy) get an ordinary HTTP client because those hosts are often private.
func clientOptions(settings Settings) *azblob.ClientOptions {
	if settings.Cloud != CloudCustom {
		return nil
	}
	return &azblob.ClientOptions{
		ClientOptions: azcore.ClientOptions{
			Transport: &http.Client{Transport: http.DefaultTransport},
		},
	}
}

// prefix keeps caller paths under the configured prefix. path.Join would let
// "foo/../../secret" escape the prefix, so a joined path that is not the
// prefix or a descendant falls back to the prefix plus the base name.
func (b *Backend) prefix(p string) string {
	joined := path.Join(b.pathPrefix, p)
	if b.pathPrefix == "" {
		return joined
	}
	cleanPrefix := strings.TrimSuffix(path.Clean(b.pathPrefix), "/")
	if joined == cleanPrefix || strings.HasPrefix(joined, cleanPrefix+"/") {
		return joined
	}
	return path.Join(cleanPrefix, path.Base(p))
}

func (b *Backend) blobClient(p string) *blob.Client {
	return b.client.ServiceClient().NewContainerClient(b.container).NewBlobClient(b.prefix(p))
}

func (b *Backend) blockBlobClient(p string) *blockblob.Client {
	return b.client.ServiceClient().NewContainerClient(b.container).NewBlockBlobClient(b.prefix(p))
}

func (b *Backend) containerClient() *container.Client {
	return b.client.ServiceClient().NewContainerClient(b.container)
}

func (b *Backend) TestConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	_, err := b.containerClient().GetProperties(ctx, nil)
	if err == nil {
		return nil
	}
	if bloberror.HasCode(err, bloberror.ContainerNotFound) {
		return fmt.Errorf("azure container %q does not exist: %w", b.container, err)
	}
	if isAuthError(err) {
		return fmt.Errorf("unable to authenticate against azure blob storage: %w", err)
	}
	return fmt.Errorf("unable to connect to azure blob storage: %w", err)
}

func (b *Backend) Reader(p string) (filestore.ReadCloseSeeker, error) {
	// The timeout covers opening the blob. The returned reader is cancelled
	// only when the caller Closes it, so a download that runs longer than
	// b.timeout is not cut off mid-stream.
	propsCtx, propsCancel := context.WithTimeout(context.Background(), b.timeout)
	defer propsCancel()
	client := b.blobClient(p)
	props, err := client.GetProperties(propsCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %q: %w", p, err)
	}
	if props.ContentLength == nil {
		return nil, fmt.Errorf("missing content length for %q", p)
	}
	return newRangeReader(client, *props.ContentLength), nil
}

func (b *Backend) ReadFile(p string) ([]byte, error) {
	r, err := b.Reader(p)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (b *Backend) FileExists(p string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	_, err := b.blobClient(p).GetProperties(ctx, nil)
	if err != nil {
		if bloberror.HasCode(err, bloberror.BlobNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("unable to check existence of %q: %w", p, err)
	}
	return true, nil
}

func (b *Backend) FileSize(p string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	props, err := b.blobClient(p).GetProperties(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("unable to get size of %q: %w", p, err)
	}
	if props.ContentLength == nil {
		return 0, nil
	}
	return *props.ContentLength, nil
}

func (b *Backend) FileModTime(p string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	props, err := b.blobClient(p).GetProperties(ctx, nil)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to get modification time of %q: %w", p, err)
	}
	if props.LastModified == nil {
		return time.Time{}, nil
	}
	return *props.LastModified, nil
}

// CopyFile starts a server-side copy and waits until Azure reports success.
// Copies inside one account are usually synchronous, but the API is not.
func (b *Backend) CopyFile(oldPath, newPath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	src := b.blobClient(oldPath).URL()
	dst := b.blockBlobClient(newPath)
	if _, err := dst.StartCopyFromURL(ctx, src, nil); err != nil {
		return fmt.Errorf("unable to copy %q to %q: %w", oldPath, newPath, err)
	}
	for {
		props, err := dst.GetProperties(ctx, nil)
		if err != nil {
			return fmt.Errorf("unable to read copy status for %q: %w", newPath, err)
		}
		if props.CopyStatus == nil {
			return nil
		}
		switch *props.CopyStatus {
		case blob.CopyStatusTypeSuccess:
			return nil
		case blob.CopyStatusTypeFailed, blob.CopyStatusTypeAborted:
			desc := ""
			if props.CopyStatusDescription != nil {
				desc = *props.CopyStatusDescription
			}
			return fmt.Errorf("azure copy from %q to %q ended in status %q: %q", oldPath, newPath, *props.CopyStatus, desc)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("azure copy from %q to %q did not complete in time: %w", oldPath, newPath, ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (b *Backend) MoveFile(oldPath, newPath string) error {
	if err := b.CopyFile(oldPath, newPath); err != nil {
		return err
	}
	return b.RemoveFile(oldPath)
}

func (b *Backend) WriteFile(fr io.Reader, p string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	return b.write(ctx, fr, p)
}

// write always stages blocks and commits a block list. A single-shot Put Blob
// leaves no committed block list, and a later append would replace the blob
// with only the new bytes.
func (b *Backend) write(ctx context.Context, fr io.Reader, p string) (int64, error) {
	bb := b.blockBlobClient(p)
	blockIDs, total, err := stageBlocks(ctx, bb, fr, p, azureBlockSize)
	if err != nil {
		return 0, err
	}
	if len(blockIDs) == 0 {
		id, idErr := newBlockID()
		if idErr != nil {
			return 0, fmt.Errorf("failed to generate azure block id: %w", idErr)
		}
		if _, sbErr := bb.StageBlock(ctx, id, nopCloser{bytes.NewReader(nil)}, nil); sbErr != nil {
			return 0, fmt.Errorf("unable to stage empty block for %q: %w", p, sbErr)
		}
		blockIDs = append(blockIDs, id)
	}
	if _, err := bb.CommitBlockList(ctx, blockIDs, nil); err != nil {
		return 0, fmt.Errorf("unable to commit block list for %q: %w", p, err)
	}
	return total, nil
}

func (b *Backend) AppendFile(fr io.Reader, p string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	store := blobAppendStore{blocks: b.blockBlobClient(p), blob: b.blobClient(p), path: p}
	return commitAppend(ctx, store, fr, p)
}

func (b *Backend) RemoveFile(p string) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	_, err := b.blobClient(p).Delete(ctx, nil)
	if err != nil && !bloberror.HasCode(err, bloberror.BlobNotFound) {
		return fmt.Errorf("unable to remove file %q: %w", p, err)
	}
	return nil
}

func (b *Backend) ListDirectory(p string) ([]string, error) {
	return b.list(p, true)
}

func (b *Backend) ListDirectoryRecursively(p string) ([]string, error) {
	return b.list(p, false)
}

func (b *Backend) list(p string, hierarchy bool) ([]string, error) {
	prefix := b.prefix(p)
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	var entries []string
	if hierarchy {
		pager := b.containerClient().NewListBlobsHierarchyPager("/", &container.ListBlobsHierarchyOptions{Prefix: &prefix})
		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, fmt.Errorf("unable to list directory %q: %w", p, err)
			}
			for _, item := range page.Segment.BlobItems {
				if item.Name != nil {
					entries = append(entries, b.stripPrefix(*item.Name))
				}
			}
			for _, item := range page.Segment.BlobPrefixes {
				if item.Name == nil {
					continue
				}
				name := strings.TrimSuffix(b.stripPrefix(*item.Name), "/")
				entries = append(entries, name)
			}
		}
		return entries, nil
	}

	pager := b.containerClient().NewListBlobsFlatPager(&container.ListBlobsFlatOptions{Prefix: &prefix})
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("unable to list directory %q recursively: %w", p, err)
		}
		for _, item := range page.Segment.BlobItems {
			if item.Name != nil {
				entries = append(entries, b.stripPrefix(*item.Name))
			}
		}
	}
	return entries, nil
}

func (b *Backend) stripPrefix(name string) string {
	name = strings.TrimPrefix(name, b.pathPrefix)
	return strings.TrimPrefix(name, "/")
}

func (b *Backend) RemoveDirectory(p string) error {
	files, err := b.ListDirectoryRecursively(p)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := b.RemoveFile(f); err != nil {
			return err
		}
	}
	return nil
}

func stageBlocks(ctx context.Context, bb *blockblob.Client, fr io.Reader, p string, blockSize int) ([]string, int64, error) {
	if blockSize <= 0 {
		blockSize = azureBlockSize
	}
	buf := make([]byte, blockSize)
	var ids []string
	var total int64
	for {
		n, err := io.ReadFull(fr, buf)
		if n > 0 {
			id, idErr := newBlockID()
			if idErr != nil {
				return nil, 0, fmt.Errorf("failed to generate azure block id: %w", idErr)
			}
			if _, sbErr := bb.StageBlock(ctx, id, nopCloser{bytes.NewReader(buf[:n])}, nil); sbErr != nil {
				return nil, 0, fmt.Errorf("unable to stage block for %q: %w", p, sbErr)
			}
			ids = append(ids, id)
			total += int64(n)
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("failed to read input: %w", err)
		}
	}
	return ids, total, nil
}

// nopCloser is what StageBlock requires: a ReadSeekCloser. Closing it must
// not close the caller's reader.
type nopCloser struct {
	io.ReadSeeker
}

func (n nopCloser) Close() error { return nil }

func newBlockID() (string, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(u[:]), nil
}

func isAuthError(err error) bool {
	return bloberror.HasCode(err, bloberror.AuthenticationFailed) ||
		bloberror.HasCode(err, bloberror.AuthorizationFailure) ||
		bloberror.HasCode(err, bloberror.InvalidAuthenticationInfo)
}

// cleanPathPrefix makes the stored prefix match the key prefix() writes.
// path.Clean("./") is ".", and a "." prefix then rejects every nested path
// and keeps only the base name, so two index.json files overwrite each other.
func cleanPathPrefix(prefix string) string {
	if prefix == "" {
		return ""
	}
	cleaned := path.Clean(prefix)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

// appendStore is the blob operations AppendFile needs. Tests substitute a
// fake so a lost commit can be retried without Azure.
type appendStore interface {
	blockList(ctx context.Context) (ids []string, etag azcore.ETag, contentLength int64, err error)
	stage(ctx context.Context, r io.Reader, blockSize int) (ids []string, n int64, err error)
	open(ctx context.Context, etag azcore.ETag) (io.ReadCloser, error)
	commit(ctx context.Context, ids []string, etag azcore.ETag) error
}

type blobAppendStore struct {
	blocks *blockblob.Client
	blob   *blob.Client
	path   string
}

func (s blobAppendStore) blockList(ctx context.Context) ([]string, azcore.ETag, int64, error) {
	resp, err := s.blocks.GetBlockList(ctx, blockblob.BlockListTypeCommitted, nil)
	if err != nil {
		return nil, "", 0, err
	}
	var ids []string
	if resp.CommittedBlocks != nil {
		for _, blk := range resp.CommittedBlocks {
			if blk.Name != nil {
				ids = append(ids, *blk.Name)
			}
		}
	}
	if resp.ETag == nil {
		return nil, "", 0, fmt.Errorf("azure blob %q has no etag", s.path)
	}
	var length int64
	if resp.BlobContentLength != nil {
		length = *resp.BlobContentLength
	}
	return ids, *resp.ETag, length, nil
}

func (s blobAppendStore) stage(ctx context.Context, r io.Reader, blockSize int) ([]string, int64, error) {
	return stageBlocks(ctx, s.blocks, r, s.path, blockSize)
}

func (s blobAppendStore) open(ctx context.Context, etag azcore.ETag) (io.ReadCloser, error) {
	match := etag
	resp, err := s.blob.DownloadStream(ctx, &blob.DownloadStreamOptions{
		AccessConditions: &blob.AccessConditions{
			ModifiedAccessConditions: &blob.ModifiedAccessConditions{IfMatch: &match},
		},
	})
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("azure blob %q download has no body", s.path)
	}
	return resp.Body, nil
}

func (s blobAppendStore) commit(ctx context.Context, ids []string, etag azcore.ETag) error {
	match := etag
	_, err := s.blocks.CommitBlockList(ctx, ids, &blockblob.CommitBlockListOptions{
		AccessConditions: &blob.AccessConditions{
			ModifiedAccessConditions: &blob.ModifiedAccessConditions{IfMatch: &match},
		},
	})
	return err
}

// commitAppend stages the new bytes once, then commits them onto the current
// block list. The commit is conditional on the etag from that read. A 412
// means another writer committed in between, so the list is read again and
// the same staged blocks are committed onto the new list. Near Azure's
// 50,000-block cap the existing bytes are restaged into fewer blocks first.
func commitAppend(ctx context.Context, store appendStore, fr io.Reader, p string) (int64, error) {
	var newIDs []string
	var total int64
	staged := false
	var lastErr error
	for attempt := 0; attempt < appendCommitAttempts; attempt++ {
		existing, etag, contentLength, err := store.blockList(ctx)
		if err != nil {
			return 0, fmt.Errorf("unable to find file %q to append data: %w", p, err)
		}
		if len(existing) == 0 && contentLength > 0 {
			return 0, fmt.Errorf("refusing to append to %q: blob has content but no committed block list", p)
		}
		if !staged {
			newIDs, total, err = store.stage(ctx, fr, azureBlockSize)
			if err != nil {
				return 0, err
			}
			staged = true
		}
		ids := make([]string, 0, len(existing)+len(newIDs))
		ids = append(ids, existing...)
		ids = append(ids, newIDs...)
		if len(ids) >= azureMaxCommittedBlocks {
			ids, err = consolidatedIDs(ctx, store, p, contentLength, etag, newIDs)
			if err != nil {
				if isAppendConflict(err) {
					lastErr = err
					continue
				}
				return 0, err
			}
		}
		if err = store.commit(ctx, ids, etag); err != nil {
			if isAppendConflict(err) {
				lastErr = err
				continue
			}
			return 0, fmt.Errorf("unable to commit block list for %q: %w", p, err)
		}
		return total, nil
	}
	return 0, fmt.Errorf("unable to commit block list for %q after concurrent updates: %w", p, lastErr)
}

// consolidatedIDs restages the blob's current bytes so the commit that adds
// newIDs stays under Azure's block cap. The download is conditional on etag;
// a mismatch is a conflict and the caller reads the list again.
func consolidatedIDs(ctx context.Context, store appendStore, p string, contentLength int64, etag azcore.ETag, newIDs []string) ([]string, error) {
	blockSize, err := consolidateBlockSize(contentLength, len(newIDs))
	if err != nil {
		return nil, fmt.Errorf("unable to append to %q: %w", p, err)
	}
	body, err := store.open(ctx, etag)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	ids, n, err := store.stage(ctx, body, blockSize)
	if err != nil {
		return nil, err
	}
	if n != contentLength {
		return nil, fmt.Errorf("unable to consolidate %q: read %d bytes, blob length is %d", p, n, contentLength)
	}
	combined := append(ids, newIDs...)
	if len(combined) >= azureMaxCommittedBlocks {
		return nil, fmt.Errorf("unable to append to %q: consolidating %d bytes still needs %d blocks", p, contentLength, len(combined))
	}
	return combined, nil
}

// consolidateBlockSize is the chunk size that keeps the rewritten blob, plus
// the blocks about to be appended, under the committed-block cap.
func consolidateBlockSize(contentLength int64, newBlocks int) (int, error) {
	room := azureMaxCommittedBlocks - newBlocks
	if room < 1 {
		return 0, fmt.Errorf("append of %d blocks would exceed the %d block limit", newBlocks, azureMaxCommittedBlocks)
	}
	if contentLength <= 0 {
		return azureBlockSize, nil
	}
	needed := (contentLength + int64(room) - 1) / int64(room)
	if needed > azureConsolidateBlockBytes {
		return 0, fmt.Errorf("blob of %d bytes cannot be consolidated under %d blocks", contentLength, azureMaxCommittedBlocks)
	}
	if needed < int64(azureBlockSize) {
		return azureBlockSize, nil
	}
	return int(needed), nil
}

func isAppendConflict(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errAppendConflict) || bloberror.HasCode(err, bloberror.ConditionNotMet) {
		return true
	}
	var respErr *azcore.ResponseError
	return errors.As(err, &respErr) && respErr.StatusCode == http.StatusPreconditionFailed
}
