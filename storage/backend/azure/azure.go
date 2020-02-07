package azure

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/meltwater/drone-cache/storage/backend"

	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type azureBackend struct {
	logger       log.Logger
	cfg          backend.AzureConfig
	containerURL azblob.ContainerURL
}

// New creates an AzureBlob backend.
func New(l log.Logger, cfgs backend.Configs) (backend.Backend, error) {
	level.Warn(l).Log("msg", "using azure blob as backend")
	l = log.With(l, "backend", backend.Azure)
	c := cfgs.Azure

	// 1. From the Azure portal, get your storage account name and key and set environment variables.
	accountName, accountKey := c.AccountName, c.AccountKey
	if len(accountName) == 0 || len(accountKey) == 0 {
		return nil, fmt.Errorf("either the AZURE_ACCOUNT_NAME or AZURE_ACCOUNT_KEY environment variable is not set")
	}

	// 2. Create a default request pipeline using your storage account name and account key.
	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("azure, invalid credentials: %w", err)
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

	return &azureBackend{cfg: c, containerURL: containerURL}, nil
}

// Get TODO
func (c *azureBackend) Get(ctx context.Context, p string) (io.ReadCloser, error) {
	blobURL := c.containerURL.NewBlockBlobURL(p)

	downloadResponse, err := blobURL.Download(ctx, 0, azblob.CountToEnd, azblob.BlobAccessConditions{}, false)
	if err != nil {
		return nil, fmt.Errorf("get the object %w", err)
	}

	bodyStream := downloadResponse.Body(azblob.RetryReaderOptions{MaxRetryRequests: c.cfg.MaxRetryRequests})

	return bodyStream, nil
}

// Put uploads the contents of the io.ReadSeeker
func (c *azureBackend) Put(ctx context.Context, p string, src io.ReadSeeker) error {
	blobURL := c.containerURL.NewBlockBlobURL(p)

	c.logger.Log("msg", "uploading the file with blob", "name", p)

	_, err := blobURL.Upload(ctx, src, azblob.BlobHTTPHeaders{}, azblob.Metadata{}, azblob.BlobAccessConditions{})
	if err != nil {
		return fmt.Errorf("put the object %w", err)
	}

	return nil
}
