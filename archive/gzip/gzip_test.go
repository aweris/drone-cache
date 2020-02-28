package gzip

import (
	"compress/flate"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/meltwater/drone-cache/archive/tar"
	"github.com/meltwater/drone-cache/test"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	file, fileClean := test.CreateTempFile(t, "tar_create", []byte("hello\ndrone!\n")) // 13 bytes
	t.Cleanup(fileClean)

	symlink := filepath.Join(filepath.Dir(file), "symlink.testfile")
	err := os.Symlink(file, symlink)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	t.Cleanup(func() { os.Remove(symlink) })

	dir, dirClean := test.CreateTempFilesInDir(t, "tar_create", []byte("hello\ngo!\n")) // 10 bytes
	t.Cleanup(dirClean)

	dstDir, dstDirClean := test.CreateTempDir(t, "tar_create_destination")
	t.Cleanup(dstDirClean)

	extDir, extDirClean := test.CreateTempDir(t, "tar_create_extracted")
	t.Cleanup(extDirClean)

	for _, tc := range []struct {
		name    string
		tgz     *gzipArchive
		srcs    []string
		written int64
		err     error
	}{
		{
			name:    "empty mount paths",
			tgz:     New(log.NewNopLogger(), true, flate.DefaultCompression),
			srcs:    []string{},
			written: 0,
			err:     nil,
		},
		{
			name: "non-existing mount paths",
			tgz:  New(log.NewNopLogger(), true, flate.DefaultCompression),
			srcs: []string{
				"iamnotexists",
				"metoo",
			},
			written: 0,
			err:     tar.ErrSourceNotReachable, // os.ErrNotExist || os.ErrPermission
		},
		{
			name: "existing mount paths",
			tgz:  New(log.NewNopLogger(), true, flate.DefaultCompression),
			srcs: []string{
				file,
				dir,
			},
			written: 43, // 3 x tmpfile in dir, 1 tmpfile
			err:     nil,
		},
		{
			name: "existing mount paths with symbolic links",
			tgz:  New(log.NewNopLogger(), true, flate.DefaultCompression),
			srcs: []string{
				file,
				dir,
				symlink,
			},
			written: 43,
			err:     nil,
		},
	} {
		tc := tc // NOTE: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			archivePath := filepath.Join(dstDir, filepath.Clean(tc.name+".tar.gz"))

			written, err := create(tc.tgz, tc.srcs, archivePath)
			if err != nil {
				test.Expected(t, err, tc.err)
				return
			}

			test.Exists(t, archivePath)
			test.Assert(t, written == tc.written,
				"case %q: written bytes got %d want %v", tc.name, written, tc.written)

			test.Ok(t, test.ExtractArchive(archivePath, extDir))
			test.EqualDirs(t, extDir, tc.srcs)
		})
	}
}

func TestExtract(t *testing.T) {
	t.Parallel()

	file, fileClean := test.CreateTempFile(t, "tar_extract", []byte("hello\ndrone!\n")) // 13 bytes
	t.Cleanup(fileClean)

	symlink := filepath.Join(filepath.Dir(file), "symlink.testfile")
	err := os.Symlink(file, symlink)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	t.Cleanup(func() { os.Remove(symlink) })

	dir, dirClean := test.CreateTempFilesInDir(t, "tar_extract", []byte("hello\ngo!\n")) // 10 bytes
	t.Cleanup(dirClean)

	arcDir, arcDirClean := test.CreateTempDir(t, "tar_extract_archive")
	t.Cleanup(arcDirClean)

	archivePath := filepath.Join(arcDir, "test.tar.gz")
	if err := test.CreateArchive([]string{file, dir}, archivePath); err != nil {
		t.Fatalf("test archive not created: %v", err)
	}

	emptyArchivePath := filepath.Join(arcDir, "empty_test.tar.gz")
	if err := test.CreateArchive([]string{}, emptyArchivePath); err != nil {
		t.Fatalf("test archive not created: %v", err)
	}

	badArchivePath := filepath.Join(arcDir, "bad_test.tar.gz")
	if err := ioutil.WriteFile(badArchivePath, []byte("hello\ndrone\n"), 0644); err != nil {
		t.Fatalf("test archive not created: %v", err)
	}

	for _, tc := range []struct {
		name        string
		tgz         *gzipArchive
		archivePath string
		srcs        []string
		content     []byte
		written     int64
		err         error
	}{
		{
			name:        "non-existing archive",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: "iamnotexists",
			srcs:        []string{},
			content:     []byte(""),
			written:     0,
			err:         os.ErrNotExist,
		},
		{
			name:        "empty archive",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: emptyArchivePath,
			srcs:        []string{},
			content:     []byte(""),
			written:     0,
			err:         nil,
		},
		{
			name:        "bad archives",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: badArchivePath,
			srcs:        []string{},
			content:     []byte(""),
			written:     0,
			err:         tar.ErrArchiveNotReadable,
		},
		{
			name:        "existing archive",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: archivePath,
			srcs: []string{
				file,
				dir,
			},
			content: []byte(""),
			written: 43,
			err:     nil,
		},
		{
			name:        "existing archive with symbolic links",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: archivePath,
			srcs: []string{
				file,
				dir,
				symlink,
			},
			content: []byte(""),
			written: 43,
			err:     nil,
		},
	} {
		tc := tc // NOTE: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dstDir, dstDirClean := test.CreateTempDir(t, "tar_extract_extracted")
			t.Cleanup(dstDirClean)

			written, err := extract(tc.tgz, tc.archivePath, dstDir)
			if err != nil {
				test.Expected(t, err, tc.err)
				return
			}

			test.Assert(t, written == tc.written,
				"case %q: written bytes got %d want %v", tc.name, written, tc.written)

			// TODO: Parked, known issue for relative paths
			// test.EqualDirs(t, dstDir, tc.srcs)
		})
	}
}

// Helpers

func create(a *gzipArchive, srcs []string, dst string) (int64, error) {
	pr, pw := io.Pipe()
	defer pr.Close()

	done := make(chan struct{})

	var written int64
	go func(w *int64) {
		defer close(done)
		defer pw.Close()

		written, err := a.Create(srcs, pw)
		if err != nil {
			pw.CloseWithError(err)
		}

		*w = written
	}(&written)

	content, err := ioutil.ReadAll(pr)
	if err != nil {
		pr.CloseWithError(err)
		return 0, err
	}

	<-done

	if err := ioutil.WriteFile(dst, content, 0644); err != nil {
		return 0, err
	}

	return written, nil
}

func extract(a *gzipArchive, src string, dst string) (int64, error) {
	pr, pw := io.Pipe()
	defer pr.Close()

	done := make(chan struct{})

	f, err := os.Open(src)
	if err != nil {
		return 0, err
	}

	go func() {
		defer close(done)
		defer pw.Close()

		_, err = io.Copy(pw, f)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
	}()

	return a.Extract(dst, pr)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		return !os.IsNotExist(err)
	}

	return true
}
