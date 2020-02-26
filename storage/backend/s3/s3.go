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
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type s3Backend struct {
	bucket     string
	acl        string
	encryption string
	client     *s3.S3
}

// New creates an S3 backend.
func New(l log.Logger, c Config, debug bool) (*s3Backend, error) {
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

	return &s3Backend{
		bucket:     c.Bucket,
		acl:        c.ACL,
		encryption: c.Encryption,
		client:     client,
	}, nil
}

// Get returns an io.Reader for reading the contents of the file.
func (c *s3Backend) Get(ctx context.Context, p string) (io.ReadCloser, error) {
	out, err := c.client.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(p),
	})
	if err != nil {
		return nil, fmt.Errorf("get the object %w", err)
	}

	return out.Body, nil
}

// Put uploads the contents of the io.ReadSeeker.
func (c *s3Backend) Put(ctx context.Context, p string, src io.ReadSeeker) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(p),
		ACL:    aws.String(c.acl),
		Body:   src,
	}
	if c.encryption != "" {
		in.ServerSideEncryption = aws.String(c.encryption)
	}

	// TODO: !! Implement streaming upload.
	// storage/backend/s3/s3.go:74:3: cannot use src (type io.Reader) as type io.ReadSeeker in field value:
	// io.Reader does not implement io.ReadSeeker (missing Seek method)

	if _, err := c.client.PutObject(in); err != nil {
		return fmt.Errorf("put the object %w", err)
	}

	return nil
}
