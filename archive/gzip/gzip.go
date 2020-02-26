package gzip

import (
	"compress/gzip"
	"fmt"
	"io"

	"github.com/meltwater/drone-cache/archive/tar"

	"github.com/go-kit/kit/log"
)

type gzipArchive struct {
	logger log.Logger

	compressionLevel int
	skipSymlinks     bool
}

// New creates an archive that uses the .tar.gz file format.
func New(logger log.Logger, compressionLevel int, skipSymlinks bool) *gzipArchive {
	return &gzipArchive{logger, compressionLevel, skipSymlinks}
}

// Create writes content of the given source to an archive, returns written bytes.
func (a *gzipArchive) Create(src string, w io.Writer) (int64, error) {
	gw, err := gzip.NewWriterLevel(w, a.compressionLevel)
	if err != nil {
		return 0, fmt.Errorf("create archive writer %w", err)
	}

	defer gw.Close()

	return tar.New(a.logger, a.skipSymlinks).Create(src, gw)
}

// Extract reads content from the given archive reader and restores it to the destination, returns written bytes.
func (a *gzipArchive) Extract(dst string, r io.Reader) (int64, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return 0, err
	}

	defer gr.Close()

	return tar.New(a.logger, a.skipSymlinks).Extract(dst, gr)
}
