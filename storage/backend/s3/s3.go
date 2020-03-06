package s3

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Backend TODO
type Backend struct {
	bucket     string
	acl        string
	encryption string
	client     *s3.S3
}

// New creates an S3 backend.
func New(l log.Logger, c Config, debug bool) (*Backend, error) {
	awsConf := &aws.Config{
		Region:           aws.String(c.Region),
		Endpoint:         &c.Endpoint,
		DisableSSL:       aws.Bool(!strings.HasPrefix(c.Endpoint, "https://")),
		S3ForcePathStyle: aws.Bool(c.PathStyle),
	}

	if c.Key != "" && c.Secret != "" {
		awsConf.Credentials = credentials.NewStaticCredentials(c.Key, c.Secret, "")
	} else {
		level.Warn(l).Log("msg", "aws key and/or Secret not provided (falling back to anonymous credentials)")
	}

	level.Debug(l).Log("msg", "s3 backend", "config", fmt.Sprintf("%#v", c))

	if debug {
		awsConf.WithLogLevel(aws.LogDebugWithHTTPBody)
	}

	client := s3.New(session.Must(session.NewSessionWithOptions(session.Options{})), awsConf)

	return &Backend{
		bucket:     c.Bucket,
		acl:        c.ACL,
		encryption: c.Encryption,
		client:     client,
	}, nil
}

// Get returns an io.Reader for reading the contents of the file.
func (c *Backend) Get(ctx context.Context, p string) (io.ReadCloser, error) {
	// downloader := s3manager.NewDownloaderWithClient(c.client)
	in := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(p),
	}

	out, err := c.client.GetObjectWithContext(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("get the object %w", err)
	}

	// if err := downloader.DownloadWithContext(ctx, io.WriterAt, in); err != nil {
	// 	return nil, fmt.Errorf("get the object %w", err)
	// }

	return out.Body, nil
}

// Put uploads the contents of the io.ReadSeeker.
func (c *Backend) Put(ctx context.Context, p string, r io.Reader) error {
	uploader := s3manager.NewUploaderWithClient(c.client)
	in := &s3manager.UploadInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(p),
		ACL:    aws.String(c.acl),
		Body:   r,
	}

	if c.encryption != "" {
		in.ServerSideEncryption = aws.String(c.encryption)
	}

	// TODO: Test!!
	if _, err := uploader.UploadWithContext(ctx, in); err != nil {
		return fmt.Errorf("put the object %w", err)
	}

	return nil
}
