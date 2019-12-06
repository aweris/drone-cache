package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// A Writer writes content of directory to an archive.
type Writer struct {
	tw *tar.Writer
	gw *gzip.Writer

	source           string
	format           string
	compressionLevel int
	skipSymlinks     bool
}

// NewWriter creates a new archive.Writer.
func NewWriter(source, format string, compressionLevel int, skipSymlinks bool) *Writer {
	return &Writer{
		source:           source,
		format:           format,
		compressionLevel: compressionLevel,
		skipSymlinks:     skipSymlinks,
	}
}

// WriteTo writes content of given directory to the given io.Writer, using specified format.
func (w *Writer) WriteTo(wrt io.Writer) (int64, error) {
	switch w.format {
	case "gzip":
		gw, err := gzip.NewWriterLevel(wrt, w.compressionLevel)
		if err != nil {
			return 0, fmt.Errorf("create archive writer %w", err)
		}

		w.gw = gw
		w.tw = tar.NewWriter(gw)
	default:
		w.tw = tar.NewWriter(wrt)
	}

	var written int64
	if err := filepath.Walk(w.source, writeToArchive(w.tw, &written, w.skipSymlinks)); err != nil {
		return written, fmt.Errorf("add all files to archive %w", err)
	}

	return written, nil
}

// Close closes the Writer.
func (w *Writer) Close() error {
	if w.tw != nil {
		return w.tw.Close()
	}

	if w.gw != nil {
		return w.gw.Close()
	}

	return nil
}

// Helpers

func writeToArchive(tw *tar.Writer, written *int64, skipSymlinks bool) func(string, os.FileInfo, error) error {
	return func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi == nil {
			return fmt.Errorf("no file info")
		}

		// Create header for Regular files and Directories
		h, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return fmt.Errorf("create header for <%s> %w", path, err)
		}

		if isSymlink(fi) {
			if skipSymlinks {
				return nil
			}

			var err error
			if h, err = createSymlinkHeader(fi, path); err != nil {
				return fmt.Errorf("create header for symbolic link %w", err)
			}
		}

		h.Name = path // to give absolute path

		if err := tw.WriteHeader(h); err != nil {
			return fmt.Errorf("write header for <%s> %w", path, err)
		}

		if fi.Mode().IsRegular() { // open and write only if it is a regular file
			n, err := writeFileToArchive(tw, path)
			*written += n
			if err != nil {
				return fmt.Errorf("write file to archive %w", err)
			}
		}

		// TODO:
		// *written += h.FileInfo().Size()
		// *written += fi.Size()

		return nil
	}
}

func createSymlinkHeader(fi os.FileInfo, path string) (*tar.Header, error) {
	lnk, err := os.Readlink(path)
	if err != nil {
		return nil, fmt.Errorf("read link <%s> %w", path, err)
	}

	h, err := tar.FileInfoHeader(fi, lnk)
	if err != nil {
		return nil, fmt.Errorf("create symlink header for <%s> %w", path, err)
	}

	return h, nil
}

func writeFileToArchive(tw io.Writer, path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open file <%s> %w", path, err)
	}
	defer f.Close()

	written, err := io.Copy(tw, f)
	if err != nil {
		return written, fmt.Errorf("copy the file <%s> data to the tarball %w", path, err)
	}

	return written, nil
}

func isSymlink(fi os.FileInfo) bool {
	return fi.Mode()&os.ModeSymlink != 0
}
