package backend

import (
	"github.com/meltwater/drone-cache/storage/backend/azure"
	"github.com/meltwater/drone-cache/storage/backend/filesystem"
	"github.com/meltwater/drone-cache/storage/backend/gcs"
	"github.com/meltwater/drone-cache/storage/backend/s3"
	"github.com/meltwater/drone-cache/storage/backend/sftp"
)

type Configs struct {
	Debug bool

	S3         s3.Config
	FileSystem filesystem.Config
	SFTP       sftp.Config
	Azure      azure.Config
	GCS        gcs.Config
}

// Config configures behavior of Backend.
type Config interface {
	Apply(*Configs)
}

type configFunc func(*Configs)

func (f configFunc) Apply(c *Configs) {
	f(c)
}

// WithDebug sets debug flag.
func WithDebug(b bool) Config {
	return configFunc(func(c *Configs) {
		c.Debug = b
	})
}

// WithS3 sets debug flag.
func WithS3(cfg s3.Config) Config {
	return configFunc(func(c *Configs) {
		c.S3 = cfg
	})
}

// WithFileSystem sets debug flag.
func WithFileSystem(cfg filesystem.Config) Config {
	return configFunc(func(c *Configs) {
		c.FileSystem = cfg
	})
}

// WithAzure sets debug flag.
func WithAzure(cfg azure.Config) Config {
	return configFunc(func(c *Configs) {
		c.Azure = cfg
	})
}

// WithSFTP sets debug flag.
func WithSFTP(cfg sftp.Config) Config {
	return configFunc(func(c *Configs) {
		c.SFTP = cfg
	})
}

// WithGCS sets debug flag.
func WithGCS(cfg gcs.Config) Config {
	return configFunc(func(c *Configs) {
		c.GCS = cfg
	})
}
