package plugin

import (
	"github.com/meltwater/drone-cache/storage/backend/azure"
	"github.com/meltwater/drone-cache/storage/backend/filesystem"
	"github.com/meltwater/drone-cache/storage/backend/gcs"
	"github.com/meltwater/drone-cache/storage/backend/s3"
	"github.com/meltwater/drone-cache/storage/backend/sftp"
)

// Config plugin-specific parameters and secrets.
type Config struct {
	ArchiveFormat    string
	Backend          string
	CacheKeyTemplate string

	// Modes
	Debug   bool
	Rebuild bool
	Restore bool

	// Optional
	SkipSymlinks     bool
	CompressionLevel int

	Mount []string

	// Backend
	S3         s3.Config
	FileSystem filesystem.Config
	SFTP       sftp.Config
	Azure      azure.Config
	GCS        gcs.Config
}
