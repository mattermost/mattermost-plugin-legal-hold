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
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
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
// A final element of ".." is rejected: path.Base leaves it unchanged, and
// joining it would resolve to the parent of the prefix.
func (b *Backend) prefix(p string) (string, error) {
	if path.Base(p) == ".." {
		return "", fmt.Errorf("path %q escapes the filestore prefix", p)
	}
	// path.Join("", "") is ".". An empty key is the container root, not a blob named ".".
	if path.Clean(b.pathPrefix) == "." && path.Clean(p) == "." {
		return "", nil
	}
	joined := path.Join(b.pathPrefix, p)
	if b.pathPrefix == "" {
		return joined, nil
	}
	cleanPrefix := strings.TrimSuffix(path.Clean(b.pathPrefix), "/")
	if joined == cleanPrefix || strings.HasPrefix(joined, cleanPrefix+"/") {
		return joined, nil
	}
	base := path.Base(p)
	if base == ".." || base == "." {
		return "", fmt.Errorf("path %q escapes the filestore prefix", p)
	}
	return path.Join(cleanPrefix, base), nil
}

func (b *Backend) blobClient(p string) (*blob.Client, error) {
	key, err := b.prefix(p)
	if err != nil {
		return nil, err
	}
	return b.client.ServiceClient().NewContainerClient(b.container).NewBlobClient(key), nil
}

func (b *Backend) blockBlobClient(p string) (*blockblob.Client, error) {
	key, err := b.prefix(p)
	if err != nil {
		return nil, err
	}
	return b.client.ServiceClient().NewContainerClient(b.container).NewBlockBlobClient(key), nil
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
	// The timeout covers opening the blob. The reader then uses the same
	// duration as an inactivity limit: it resets when bytes arrive, and it
	// cancels a body that stops sending. A download that keeps making
	// progress is not cut off.
	propsCtx, propsCancel := context.WithTimeout(context.Background(), b.timeout)
	defer propsCancel()
	client, err := b.blobClient(p)
	if err != nil {
		return nil, err
	}
	props, err := client.GetProperties(propsCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %q: %w", p, err)
	}
	if props.ContentLength == nil {
		return nil, fmt.Errorf("missing content length for %q", p)
	}
	return newRangeReader(client, *props.ContentLength, b.timeout), nil
}

func (b *Backend) ReadFile(p string) ([]byte, error) {
	r, err := b.Reader(p)
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Close() }()
	return io.ReadAll(r)
}

func (b *Backend) FileExists(p string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	client, keyErr := b.blobClient(p)
	if keyErr != nil {
		return false, keyErr
	}
	_, err := client.GetProperties(ctx, nil)
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
	client, keyErr := b.blobClient(p)
	if keyErr != nil {
		return 0, keyErr
	}
	props, err := client.GetProperties(ctx, nil)
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
	client, keyErr := b.blobClient(p)
	if keyErr != nil {
		return time.Time{}, keyErr
	}
	props, err := client.GetProperties(ctx, nil)
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
	srcClient, err := b.blobClient(oldPath)
	if err != nil {
		return err
	}
	dst, err := b.blockBlobClient(newPath)
	if err != nil {
		return err
	}
	src := srcClient.URL()
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
	bb, err := b.blockBlobClient(p)
	if err != nil {
		return 0, err
	}
	blockIDs, total, err := stageBlocks(ctx, bb, fr, p, azureBlockSize, "")
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
	blocks, err := b.blockBlobClient(p)
	if err != nil {
		return 0, err
	}
	blobs, err := b.blobClient(p)
	if err != nil {
		return 0, err
	}
	store := &blobAppendStore{blocks: blocks, blob: blobs, path: p}
	return commitAppend(ctx, store, fr, p)
}

func (b *Backend) RemoveFile(p string) error {
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()
	client, keyErr := b.blobClient(p)
	if keyErr != nil {
		return keyErr
	}
	_, err := client.Delete(ctx, nil)
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
	prefix, err := b.prefix(p)
	if err != nil {
		return nil, err
	}
	// A root list must not be scoped to "./", or blobs such as legalhold/index.json
	// are skipped and RemoveDirectory("") reports success without deleting them.
	if prefix == "." {
		prefix = ""
	} else if prefix != "" && !strings.HasSuffix(prefix, "/") {
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

func stageBlocks(ctx context.Context, bb *blockblob.Client, fr io.Reader, p string, blockSize int, leaseID string) ([]string, int64, error) {
	if blockSize <= 0 {
		blockSize = azureBlockSize
	}
	var stageOpts *blockblob.StageBlockOptions
	if leaseID != "" {
		id := leaseID
		stageOpts = &blockblob.StageBlockOptions{
			LeaseAccessConditions: &blob.LeaseAccessConditions{LeaseID: &id},
		}
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
			if _, sbErr := bb.StageBlock(ctx, id, nopCloser{bytes.NewReader(buf[:n])}, stageOpts); sbErr != nil {
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
	lock(ctx context.Context) error
	unlock(ctx context.Context) error
	blockList(ctx context.Context) (ids []string, etag azcore.ETag, contentLength int64, err error)
	stage(ctx context.Context, r io.Reader, blockSize int) (ids []string, n int64, err error)
	open(ctx context.Context, etag azcore.ETag) (io.ReadCloser, error)
	commit(ctx context.Context, ids []string, etag azcore.ETag) error
}

type blobAppendStore struct {
	blocks      *blockblob.Client
	blob        *blob.Client
	path        string
	leaseClient *lease.BlobClient
	leaseID     string
	stopRenew   context.CancelFunc
	renewDone   chan struct{}
}

func (s *blobAppendStore) lock(ctx context.Context) error {
	if s.leaseClient != nil {
		return nil
	}
	lc, err := lease.NewBlobClient(s.blob, nil)
	if err != nil {
		return fmt.Errorf("unable to lease %q: %w", s.path, err)
	}
	// 60 seconds is Azure's longest fixed lease. A renewer keeps it alive for
	// the rest of the append, including the check that the commit landed.
	if _, err = lc.AcquireLease(ctx, 60, nil); err != nil {
		return fmt.Errorf("unable to lease %q: %w", s.path, err)
	}
	if lc.LeaseID() == nil {
		return fmt.Errorf("azure lease for %q has no id", s.path)
	}
	s.leaseClient = lc
	s.leaseID = *lc.LeaseID()
	renewCtx, cancel := context.WithCancel(ctx)
	s.stopRenew = cancel
	s.renewDone = make(chan struct{})
	go func() {
		defer close(s.renewDone)
		s.renewLease(renewCtx)
	}()
	return nil
}

func (s *blobAppendStore) renewLease(ctx context.Context) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = s.leaseClient.RenewLease(ctx, nil)
		}
	}
}

func (s *blobAppendStore) unlock(ctx context.Context) error {
	if s.stopRenew != nil {
		s.stopRenew()
		s.stopRenew = nil
	}
	if s.renewDone != nil {
		<-s.renewDone
		s.renewDone = nil
	}
	if s.leaseClient == nil {
		return nil
	}
	_, err := s.leaseClient.ReleaseLease(ctx, nil)
	s.leaseClient = nil
	s.leaseID = ""
	return err
}

func (s *blobAppendStore) leaseConditions(match *azcore.ETag) *blob.AccessConditions {
	conds := &blob.AccessConditions{}
	if match != nil {
		conds.ModifiedAccessConditions = &blob.ModifiedAccessConditions{IfMatch: match}
	}
	if s.leaseID != "" {
		id := s.leaseID
		conds.LeaseAccessConditions = &blob.LeaseAccessConditions{LeaseID: &id}
	}
	if conds.ModifiedAccessConditions == nil && conds.LeaseAccessConditions == nil {
		return nil
	}
	return conds
}

func (s *blobAppendStore) blockList(ctx context.Context) ([]string, azcore.ETag, int64, error) {
	resp, err := s.blocks.GetBlockList(ctx, blockblob.BlockListTypeCommitted, &blockblob.GetBlockListOptions{
		AccessConditions: s.leaseConditions(nil),
	})
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

func (s *blobAppendStore) stage(ctx context.Context, r io.Reader, blockSize int) ([]string, int64, error) {
	return stageBlocks(ctx, s.blocks, r, s.path, blockSize, s.leaseID)
}

func (s *blobAppendStore) open(ctx context.Context, etag azcore.ETag) (io.ReadCloser, error) {
	match := etag
	resp, err := s.blob.DownloadStream(ctx, &blob.DownloadStreamOptions{
		AccessConditions: s.leaseConditions(&match),
	})
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return nil, fmt.Errorf("azure blob %q download has no body", s.path)
	}
	return resp.Body, nil
}

func (s *blobAppendStore) commit(ctx context.Context, ids []string, etag azcore.ETag) error {
	match := etag
	_, err := s.blocks.CommitBlockList(ctx, ids, &blockblob.CommitBlockListOptions{
		AccessConditions: s.leaseConditions(&match),
	})
	return err
}

// commitAppend commits the new bytes onto the current block list. The commit
// is conditional on the etag from that read. A 412 means another writer
// committed first. Azure drops uncommitted blocks that the winning list omits,
// so the input is kept and staged again instead of reusing the old block IDs.
// A list longer than 50,000 blocks is compacted first. Exactly 50,000 is valid.
func commitAppend(ctx context.Context, store appendStore, fr io.Reader, p string) (int64, error) {
	payload, err := io.ReadAll(fr)
	if err != nil {
		return 0, fmt.Errorf("failed to read input: %w", err)
	}
	// The lease is held through the commit and the confirmation read. Another
	// append cannot consolidate this blob and replace the block IDs before we
	// know whether a lost response still landed.
	if err = store.lock(ctx); err != nil {
		return 0, fmt.Errorf("unable to lock %q for append: %w", p, err)
	}
	defer func() {
		_ = store.unlock(context.Background())
	}()
	var lastErr error
	for range appendCommitAttempts {
		existing, etag, contentLength, err := store.blockList(ctx)
		if err != nil {
			return 0, fmt.Errorf("unable to find file %q to append data: %w", p, err)
		}
		if len(existing) == 0 && contentLength > 0 {
			return 0, fmt.Errorf("refusing to append to %q: blob has content but no committed block list", p)
		}
		newIDs, total, err := store.stage(ctx, bytes.NewReader(payload), azureBlockSize)
		if err != nil {
			return 0, err
		}
		ids := make([]string, 0, len(existing)+len(newIDs))
		ids = append(ids, existing...)
		ids = append(ids, newIDs...)
		if len(ids) > azureMaxCommittedBlocks {
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
				landed, checkErr := appendAlreadyCommitted(ctx, store, newIDs)
				if checkErr != nil {
					return 0, checkErr
				}
				if landed {
					return total, nil
				}
				lastErr = err
				continue
			}
			return 0, fmt.Errorf("unable to commit block list for %q: %w", p, err)
		}
		return total, nil
	}
	return 0, fmt.Errorf("unable to commit block list for %q after concurrent updates: %w", p, lastErr)
}

// appendAlreadyCommitted reports whether the block IDs from this attempt are
// already in the committed list. A 412 after CommitBlockList can mean the
// commit landed and the response was lost. Those IDs are still committed.
// A competing writer omits them, so they are absent and the append must be staged again.
func appendAlreadyCommitted(ctx context.Context, store appendStore, newIDs []string) (bool, error) {
	if len(newIDs) == 0 {
		return false, nil
	}
	committed, _, _, err := store.blockList(ctx)
	if err != nil {
		return false, fmt.Errorf("unable to confirm append commit: %w", err)
	}
	have := make(map[string]struct{}, len(committed))
	for _, id := range committed {
		have[id] = struct{}{}
	}
	for _, id := range newIDs {
		if _, ok := have[id]; !ok {
			return false, nil
		}
	}
	return true, nil
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
	defer func() { _ = body.Close() }()
	ids, n, err := store.stage(ctx, body, blockSize)
	if err != nil {
		return nil, err
	}
	if n != contentLength {
		return nil, fmt.Errorf("unable to consolidate %q: read %d bytes, blob length is %d", p, n, contentLength)
	}
	ids = append(ids, newIDs...)
	if len(ids) > azureMaxCommittedBlocks {
		return nil, fmt.Errorf("unable to append to %q: consolidating %d bytes still needs %d blocks", p, contentLength, len(ids))
	}
	return ids, nil
}

// consolidateBlockSize is the chunk size that keeps the rewritten blob, plus
// the blocks about to be appended, under the committed-block cap.
func consolidateBlockSize(contentLength int64, newBlocks int) (int, error) {
	if newBlocks > azureMaxCommittedBlocks {
		return 0, fmt.Errorf("append of %d blocks would exceed the %d block limit", newBlocks, azureMaxCommittedBlocks)
	}
	room := azureMaxCommittedBlocks - newBlocks
	if contentLength <= 0 {
		return azureBlockSize, nil
	}
	if room < 1 {
		return 0, fmt.Errorf("append of %d blocks leaves no room to keep %d existing bytes under the %d block limit", newBlocks, contentLength, azureMaxCommittedBlocks)
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
