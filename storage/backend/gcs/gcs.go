package gcs

import (
	"context"
	"fmt"
	"io"

	gcstorage "cloud.google.com/go/storage"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"google.golang.org/api/option"
)

// gcsBackend is an Cloud Storage implementation of the Backend.
type gcsBackend struct {
	bucket     string
	acl        string
	encryption string
	client     *gcstorage.Client
}

// New creates a Google Cloud Storage backend.
func New(l log.Logger, c Config) (*gcsBackend, error) {
	var opts []option.ClientOption
	if c.APIKey != "" {
		opts = append(opts, option.WithAPIKey(c.APIKey))
	}

	if c.Endpoint != "" {
		opts = append(opts, option.WithEndpoint(c.Endpoint))
	}

	level.Debug(l).Log("msg", "gc storage backend", "config", fmt.Sprintf("%+v", c))

	client, err := gcstorage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return &gcsBackend{
		bucket:     c.Bucket,
		acl:        c.ACL,
		encryption: c.Encryption,
		client:     client,
	}, nil
}

// Get returns an io.Reader for reading the contents of the file.
func (c *gcsBackend) Get(ctx context.Context, p string) (io.ReadCloser, error) {
	bkt := c.client.Bucket(c.bucket)
	obj := bkt.Object(p)

	if c.encryption != "" {
		obj = obj.Key([]byte(c.encryption))
	}

	return obj.NewReader(ctx)
}

// Put uploads the contents of the io.Reader.
func (c *gcsBackend) Put(ctx context.Context, p string, src io.Reader) error {
	bkt := c.client.Bucket(c.bucket)

	obj := bkt.Object(p)
	if c.encryption != "" {
		obj = obj.Key([]byte(c.encryption))
	}

	w := obj.NewWriter(ctx)
	defer w.Close() // TODO:
	_, err := io.Copy(w, src)

	return err
}
