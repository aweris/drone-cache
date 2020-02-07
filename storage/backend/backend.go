package backend

import (
	"context"
	"io"
)

type SSHAuthMethod string

const (
	FileSystem = "filesystem"
	S3         = "s3"
	SFTP       = "sftp"
	Azure      = "azure"
	GCS        = "gcs"

	SSHAuthMethodPassword      SSHAuthMethod = "PASSWORD"
	SSHAuthMethodPublicKeyFile SSHAuthMethod = "PUBLIC_KEY_FILE"
)

// Backend implements operations for caching files.
type Backend interface {
	Get(ctx context.Context, p string) (io.ReadCloser, error)
	Put(ctx context.Context, p string, rs io.ReadSeeker) error
	// List(ctx context.Context, p string) ([]FileEntry, error)
	// Delete(ctx context.Context, p string) error
}

// S3Config is a structure to store S3  backend configuration
type S3Config struct {
	// Indicates the files ACL, which should be one,
	// of the following:
	//     private
	//     public-read
	//     public-read-write
	//     authenticated-read
	//     bucket-owner-read
	//     bucket-owner-full-control
	ACL        string
	Bucket     string
	Encryption string // if not "", enables server-side encryption. valid values are: AES256, aws:kms
	Endpoint   string
	Key        string

	// us-east-1
	// us-west-1
	// us-west-2
	// eu-west-1
	// ap-southeast-1
	// ap-southeast-2
	// ap-northeast-1
	// sa-east-1
	Region string
	Secret string

	PathStyle bool // Use path style instead of domain style. Should be true for minio and false for AWS
}

// AzureConfig is a structure to store Azure backend configuration
type AzureConfig struct {
	AccountName      string
	AccountKey       string
	ContainerName    string
	BlobStorageURL   string
	Azurite          bool
	MaxRetryRequests int
}

// FileSystemConfig is a structure to store filesystem backend configuration
type FileSystemConfig struct {
	CacheRoot string
}

// GCSConfig is a structure to store Cloud Storage backend configuration
type GCSConfig struct {
	Bucket     string
	ACL        string
	Encryption string
	Endpoint   string
	APIKey     string
}

type SSHAuth struct {
	Password      string
	PublicKeyFile string
	Method        SSHAuthMethod
}

// SFTPConfig is a structure to store sftp backend configuration
type SFTPConfig struct {
	CacheRoot string
	Username  string
	Host      string
	Port      string
	Auth      SSHAuth
}
