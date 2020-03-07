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

// Backend is an Cloud Storage implementation of the Backend.
type Backend struct {
	logger log.Logger

	bucket     string
	acl        string
	encryption string
	client     *gcstorage.Client
}

// New creates a Google Cloud Storage backend.
func New(l log.Logger, c Config) (*Backend, error) {
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
		return nil, fmt.Errorf("gcs client initialization %w", err)
	}

	return &Backend{
		logger:     l,
		bucket:     c.Bucket,
		acl:        c.ACL,
		encryption: c.Encryption,
		client:     client,
	}, nil
}

// Get writes downloaded content to the given writer.
func (b *Backend) Get(ctx context.Context, p string, w io.Writer) error {
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		bkt := b.client.Bucket(b.bucket)
		obj := bkt.Object(p)

		if b.encryption != "" {
			obj = obj.Key([]byte(b.encryption))
		}

		r, err := obj.NewReader(ctx)
		if err != nil {
			errCh <- fmt.Errorf("get the object %w", err)
		}
		defer r.Close()

		_, err = io.Copy(w, r)
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
	errCh := make(chan error)
	go func() {
		defer close(errCh)

		bkt := b.client.Bucket(b.bucket)
		obj := bkt.Object(p)

		if b.encryption != "" {
			obj = obj.Key([]byte(b.encryption))
		}

		w := obj.NewWriter(ctx)
		defer w.Close()

		_, err := io.Copy(w, r)
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
