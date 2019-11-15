package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

// A Extractor extracts content from an archive and writes it to disk.
type Extractor struct {
	tr  *tar.Reader
	gzr *gzip.Reader

	format string
}

// NewExtractor creates a new archive.Extractor.
func NewExtractor(format string) *Extractor {
	return &Extractor{format: format}
}

// ExtractFrom extracts content from given reader and writes it to disk.
func (r *Extractor) ExtractFrom(rdr io.Reader) (int64, error) {
	var written int64

	tr := tar.NewReader(rdr)

	switch r.format {
	case "gzip":
		gzr, err := gzip.NewReader(rdr)
		if err != nil {
			return written, fmt.Errorf("gzip reader %w", err)
		}

		r.gzr = gzr
		r.tr = tar.NewReader(gzr)
	default:
		r.tr = tr
	}

	for {
		h, err := tr.Next()

		switch {
		case err == io.EOF: // if no more files are found return
			return written, nil
		case err != nil: // return any other error
			return written, fmt.Errorf("tar reader %w", err)
		case h == nil: // if the header is nil, skip it
			continue
		}

		switch h.Typeflag {
		case tar.TypeDir:
			if err := extractDir(h); err != nil {
				return written, err
			}

			continue
		case tar.TypeReg, tar.TypeRegA, tar.TypeChar, tar.TypeBlock, tar.TypeFifo:
			n, err := extractRegular(h, tr)
			written += n

			if err != nil {
				return written, fmt.Errorf("extract regular file %w", err)
			}

			continue
		case tar.TypeSymlink:
			if err := extractSymlink(h); err != nil {
				return written, fmt.Errorf("extract symbolic link %w", err)
			}

			continue
		case tar.TypeLink:
			if err := extractLink(h); err != nil {
				return written, fmt.Errorf("extract link %w", err)
			}

			continue
		case tar.TypeXGlobalHeader:
			continue
		default:
			return written, fmt.Errorf("extract %s, unknown type flag: %c", h.Name, h.Typeflag)
		}
	}
}

// Close closes the Extractor.
func (r *Extractor) Close() error {
	if r.gzr != nil {
		return r.gzr.Close()
	}

	return nil
}

// Helpers

func extractDir(h *tar.Header) error {
	if err := os.MkdirAll(h.Name, os.FileMode(h.Mode)); err != nil {
		return fmt.Errorf("create directory <%s> %w", h.Name, err)
	}

	return nil
}

func extractRegular(h *tar.Header, tr io.Reader) (n int64, err error) {
	f, err := os.OpenFile(h.Name, os.O_CREATE|os.O_RDWR, os.FileMode(h.Mode))
	if err != nil {
		return 0, fmt.Errorf("open extracted file for writing <%s> %w", h.Name, err)
	}
	defer f.Close()

	written, err := io.Copy(f, tr)
	if err != nil {
		return written, fmt.Errorf("copy extracted file for writing <%s> %w", h.Name, err)
	}

	return written, nil
}

func extractSymlink(h *tar.Header) error {
	if err := unlink(h.Name); err != nil {
		return fmt.Errorf("unlink <%s> %w", h.Name, err)
	}

	if err := os.Symlink(h.Linkname, h.Name); err != nil {
		return fmt.Errorf("create symbolic link <%s> %w", h.Name, err)
	}

	return nil
}

func extractLink(h *tar.Header) error {
	if err := unlink(h.Name); err != nil {
		return fmt.Errorf("unlink <%s> %w", h.Name, err)
	}

	if err := os.Link(h.Linkname, h.Name); err != nil {
		return fmt.Errorf("create hard link <%s> %w", h.Linkname, err)
	}

	return nil
}

func unlink(path string) error {
	_, err := os.Lstat(path)
	if err == nil {
		return os.Remove(path)
	}

	return nil
}
