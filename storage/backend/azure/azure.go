package azure

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// DefaultBlobMaxRetryRequests TODO
const (
	DefaultBlobMaxRetryRequests = 4

	defaultBufferSize = 3 * 1024 * 1024
	defaultMaxBuffers = 4
)

// Backend TODO
type Backend struct {
	logger log.Logger

	cfg          Config
	containerURL azblob.ContainerURL
}

// New creates an AzureBlob backend.
func New(l log.Logger, c Config) (*Backend, error) {
	// 1. From the Azure portal, get your storage account name and key and set environment variables.
	accountName, accountKey := c.AccountName, c.AccountKey
	if len(accountName) == 0 || len(accountKey) == 0 {
		return nil, fmt.Errorf("either the AZURE_ACCOUNT_NAME or AZURE_ACCOUNT_KEY environment variable is not set")
	}

	// 2. Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("azure, invalid credentials %w", err)
	}

	var azureBlobURL *url.URL

	// 3. Azurite has different URL pattern than production Azure Blob Storage.
	if c.Azurite {
		azureBlobURL, err = url.Parse(fmt.Sprintf("http://%s/%s/%s", c.BlobStorageURL, c.AccountName, c.ContainerName))
	} else {
		azureBlobURL, err = url.Parse(fmt.Sprintf("https://%s.%s/%s", c.AccountName, c.BlobStorageURL, c.ContainerName))
	}

	if err != nil {
		level.Error(l).Log("msg", "can't create url with : "+err.Error())
	}

	pipeline := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	containerURL := azblob.NewContainerURL(*azureBlobURL, pipeline)

	// 4. Always creating new container, it will throw error if it already exists.
	_, err = containerURL.Create(context.Background(), azblob.Metadata{}, azblob.PublicAccessNone)
	if err != nil {
		// TODO: Check if we need to return the error.
		level.Debug(l).Log("msg", "container already exists", "err", err)
	}

	return &Backend{logger: l, cfg: c, containerURL: containerURL}, nil
}

// Get writes downloaded content to the given writer.
func (b *Backend) Get(ctx context.Context, p string, w io.Writer) error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		blobURL := b.containerURL.NewBlockBlobURL(p)

		resp, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
		if err != nil {
			errCh <- fmt.Errorf("get the object %w", err)
		}

		rc := resp.Body(azblob.RetryReaderOptions{MaxRetryRequests: b.cfg.MaxRetryRequests})
		defer rc.Close()

		_, err = io.Copy(w, rc)
		if err != nil {
			errCh <- fmt.Errorf("copy the object %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Put uploads contents of the given reader.
func (b *Backend) Put(ctx context.Context, p string, r io.Reader) error {
	blobURL := b.containerURL.NewBlockBlobURL(p)

	b.logger.Log("msg", "uploading the file with blob", "name", p)

	// TODO: Check stream options!
	// TODO: Test!
	if _, err := azblob.UploadStreamToBlockBlob(ctx, r, blobURL,
		azblob.UploadStreamToBlockBlobOptions{
			BufferSize: defaultBufferSize,
			MaxBuffers: defaultMaxBuffers,
		},
	); err != nil {
		return fmt.Errorf("put the object %w", err)
	}

	return nil
}
