package gzip

import (
	"compress/flate"
	"compress/gzip"
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

	for _, tc := range []struct {
		name    string
		tgz     *Archive
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
			name:    "existing mount paths",
			tgz:     New(log.NewNopLogger(), true, flate.DefaultCompression),
			srcs:    exampleFileTree(t, "gzip_create"),
			written: 43, // 3 x tmpfile in dir, 1 tmpfile
			err:     nil,
		},
		{
			name:    "existing mount paths with symbolic links",
			tgz:     New(log.NewNopLogger(), false, flate.DefaultCompression),
			srcs:    exampleFileTreeWithSymlinks(t, "gzip_create_symlink"),
			written: 43,
			err:     nil,
		},
	} {
		tc := tc // NOTE: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			dstDir, dstDirClean := test.CreateTempDir(t, "gzio_create_destination")
			t.Cleanup(dstDirClean)

			extDir, extDirClean := test.CreateTempDir(t, "gzip_create_extracted")
			t.Cleanup(extDirClean)

			// Run
			archivePath := filepath.Join(dstDir, filepath.Clean(tc.name+".tar.gz"))
			written, err := create(tc.tgz, tc.srcs, archivePath)
			if err != nil {
				test.Expected(t, err, tc.err)
				return
			}

			test.Exists(t, archivePath)
			test.Assert(t, written == tc.written, "case %q: written bytes got %d want %v", tc.name, written, tc.written)

			test.Ok(t, test.ExtractArchive(archivePath, extDir))
			test.EqualDirs(t, extDir, os.TempDir(), tc.srcs)
		})
	}
}

func TestExtract(t *testing.T) {
	t.Parallel()

	// Setup
	arcDir, arcDirClean := test.CreateTempDir(t, "gzip_extract_archive")
	t.Cleanup(arcDirClean)

	files := exampleFileTree(t, "gzip_extract")

	archivePath := filepath.Join(arcDir, "test.tar.gz")
	if err := test.CreateArchive(files, archivePath); err != nil {
		t.Fatalf("test archive not created: %v", err)
	}

	filesWithSymlink := exampleFileTreeWithSymlinks(t, "gzip_extract_symlink")
	archiveWithSymlinkPath := filepath.Join(arcDir, "test_with_symlink.tar.gz")
	if err := test.CreateArchive(filesWithSymlink, archiveWithSymlinkPath); err != nil {
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
		tgz         *Archive
		archivePath string
		srcs        []string
		written     int64
		err         error
	}{
		{
			name:        "non-existing archive",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: "iamnotexists",
			srcs:        []string{},
			written:     0,
			err:         os.ErrNotExist,
		},
		{
			name:        "non-existing root destination",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: emptyArchivePath,
			srcs:        []string{},
			written:     0,
			err:         tar.ErrDestinationNotReachable,
		},
		{
			name:        "empty archive",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: emptyArchivePath,
			srcs:        []string{},
			written:     0,
			err:         nil,
		},
		{
			name:        "bad archives",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: badArchivePath,
			srcs:        []string{},
			written:     0,
			err:         gzip.ErrHeader,
		},
		{
			name:        "existing archive",
			tgz:         New(log.NewNopLogger(), true, flate.DefaultCompression),
			archivePath: archivePath,
			srcs:        files,
			written:     43,
			err:         nil,
		},
		{
			name:        "existing archive with symbolic links",
			tgz:         New(log.NewNopLogger(), false, flate.DefaultCompression),
			archivePath: archiveWithSymlinkPath,
			srcs:        filesWithSymlink,
			written:     43,
			err:         nil,
		},
	} {
		tc := tc // NOTE: https://github.com/golang/go/wiki/CommonMistakes#using-goroutines-on-loop-iterator-variables
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dstDir, dstDirClean := test.CreateTempDir(t, "gzip_extract_extracted_"+tc.name)
			t.Cleanup(dstDirClean)

			written, err := extract(tc.tgz, tc.archivePath, dstDir)
			if err != nil {
				test.Expected(t, err, tc.err)
				return
			}

			test.Assert(t, written == tc.written, "case %q: written bytes got %d want %v", tc.name, written, tc.written)
			test.EqualDirs(t, dstDir, os.TempDir(), tc.srcs)
		})
	}
}

// Helpers

func create(a *Archive, srcs []string, dst string) (int64, error) {
	pr, pw := io.Pipe()
	defer pr.Close()

	var written int64
	go func(w *int64) {
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

	if err := ioutil.WriteFile(dst, content, 0644); err != nil {
		return 0, err
	}

	return written, nil
}

func extract(a *Archive, src string, dst string) (int64, error) {
	pr, pw := io.Pipe()
	defer pr.Close()

	f, err := os.Open(src)
	if err != nil {
		return 0, err
	}

	go func() {
		defer pw.Close()

		_, err = io.Copy(pw, f)
		if err != nil {
			pw.CloseWithError(err)
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

// Fixtures

func exampleFileTree(t *testing.T, name string) []string {
	file, fileClean := test.CreateTempFile(t, name, []byte("hello\ndrone!\n")) // 13 bytes
	t.Cleanup(fileClean)

	dir, dirClean := test.CreateTempFilesInDir(t, name, []byte("hello\ngo!\n")) // 10 bytes
	t.Cleanup(dirClean)

	return []string{file, dir}
}

func exampleFileTreeWithSymlinks(t *testing.T, name string) []string {
	file, fileClean := test.CreateTempFile(t, name, []byte("hello\ndrone!\n")) // 13 bytes
	t.Cleanup(fileClean)

	symlink := filepath.Join(filepath.Dir(file), name+"_symlink.testfile")
	err := os.Symlink(file, symlink)
	if err != nil {
		t.Fatalf("link not created: %v", err)
	}
	t.Cleanup(func() { os.Remove(symlink) })

	dir, dirClean := test.CreateTempFilesInDir(t, name, []byte("hello\ngo!\n")) // 10 bytes
	t.Cleanup(dirClean)

	return []string{file, dir, symlink}
}
