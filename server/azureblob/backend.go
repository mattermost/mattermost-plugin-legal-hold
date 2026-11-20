package azureblob

import (
	"bytes"
	"context"
	"encoding/base64"
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

// azureBlockSize matches the Azure SDK UploadStream default and stays under
// the per-block REST limit. Every write goes through StageBlock so a later
// append can commit a new block list without clobbering a Put Blob.
const azureBlockSize = 4 * 1024 * 1024

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
	if settings.PathPrefix != "" {
		settings.PathPrefix = path.Clean(settings.PathPrefix)
	}
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
	ctx, cancel := context.WithCancel(context.Background())
	timer := time.AfterFunc(b.timeout, cancel)
	client := b.blobClient(p)
	props, err := client.GetProperties(ctx, nil)
	if err != nil {
		timer.Stop()
		cancel()
		return nil, fmt.Errorf("unable to read file %q: %w", p, err)
	}
	if props.ContentLength == nil {
		timer.Stop()
		cancel()
		return nil, fmt.Errorf("missing content length for %q", p)
	}
	return &rangeReader{
		ctx:        ctx,
		cancel:     cancel,
		timer:      timer,
		blobClient: client,
		size:       *props.ContentLength,
	}, nil
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
	blockIDs, total, err := stageBlocks(ctx, bb, fr, p)
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
	bb := b.blockBlobClient(p)
	listResp, err := bb.GetBlockList(ctx, blockblob.BlockListTypeCommitted, nil)
	if err != nil {
		return 0, fmt.Errorf("unable to find file %q to append data: %w", p, err)
	}
	var existingIDs []string
	if listResp.BlockList.CommittedBlocks != nil {
		for _, blk := range listResp.BlockList.CommittedBlocks {
			if blk.Name != nil {
				existingIDs = append(existingIDs, *blk.Name)
			}
		}
	}
	if len(existingIDs) == 0 {
		props, propsErr := bb.GetProperties(ctx, nil)
		if propsErr != nil {
			return 0, fmt.Errorf("unable to inspect %q before append: %w", p, propsErr)
		}
		if props.ContentLength != nil && *props.ContentLength > 0 {
			return 0, fmt.Errorf("refusing to append to %q: blob has content but no committed block list", p)
		}
	}
	newIDs, total, err := stageBlocks(ctx, bb, fr, p)
	if err != nil {
		return 0, err
	}
	if _, err := bb.CommitBlockList(ctx, append(existingIDs, newIDs...), nil); err != nil {
		return 0, fmt.Errorf("unable to commit block list for %q: %w", p, err)
	}
	return total, nil
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

func stageBlocks(ctx context.Context, bb *blockblob.Client, fr io.Reader, p string) ([]string, int64, error) {
	buf := make([]byte, azureBlockSize)
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
