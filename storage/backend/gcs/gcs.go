package backend

import (
	"context"
	"io"

	"github.com/meltwater/drone-cache/storage"

	gcstorage "cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// gcsBackend is an Cloud Storage implementation of the Backend
type gcsBackend struct {
	bucket     string
	acl        string
	encryption string
	client     *gcstorage.Client
}

// newGCS returns a new Cloud Storage remote Backend implemented
func newGCS(bucket, acl, encryption string, opts ...option.ClientOption) (storage.Backend, error) {
	ctx := context.Background()
	client, err := gcstorage.NewClient(ctx, opts...)

	if err != nil {
		return nil, err
	}

	return &gcsBackend{
		bucket:     bucket,
		acl:        acl,
		encryption: encryption,
		client:     client,
	}, nil
}

// Get returns an io.Reader for reading the contents of the file
func (c *gcsBackend) Get(p string) (io.ReadCloser, error) {
	bkt := c.client.Bucket(c.bucket)
	obj := bkt.Object(p)

	if c.encryption != "" {
		obj = obj.Key([]byte(c.encryption))
	}

	return obj.NewReader(context.TODO())
}

// Put uploads the contents of the io.ReadSeeker
func (c *gcsBackend) Put(p string, src io.ReadSeeker) error {
	bkt := c.client.Bucket(c.bucket)

	obj := bkt.Object(p)
	if c.encryption != "" {
		obj = obj.Key([]byte(c.encryption))
	}

	w := obj.NewWriter(context.TODO())
	_, err := io.Copy(w, src)

	return err
}
