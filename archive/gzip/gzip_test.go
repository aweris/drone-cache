package gzip

import (
	"compress/flate"
	"errors"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/meltwater/drone-cache/test"
)

func TestCreate(t *testing.T) {
	t.Parallel()

	filePath, fileClean := test.CreateTempFile(t, "gzip_create", []byte("hello\ndrone!\n"))
	t.Cleanup(fileClean)

	srcDir, srcDirClean := test.CreateTempFilesInDir(t, "gzip_create", []byte("hello\ngo!\n"))
	t.Cleanup(srcDirClean)

	dstDir, dstDirClean := test.CreateTempDir(t, "gzip_create")
	t.Cleanup(dstDirClean)

	for _, tc := range []struct {
		name       string
		tgz        *gzipArchive
		srcs       []string
		expWritten int64
		err        error
	}{
		{
			name:       "empty mount paths",
			tgz:        New(log.NewNopLogger(), flate.DefaultCompression, true),
			srcs:       []string{},
			expWritten: 0,
			err:        nil,
		},
		{
			name: "non-existing mount paths",
			tgz:  New(log.NewNopLogger(), flate.DefaultCompression, true),
			srcs: []string{
				"iamnotexists",
				"metoo",
			},
			expWritten: 0,
			err:        nil,
		},
		{
			name: "existing mount paths",
			tgz:  New(log.NewNopLogger(), flate.DefaultCompression, true),
			srcs: []string{
				filePath,
				srcDir,
			},
			expWritten: 13,
			err:        nil,
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			written, err := create(tc.tgz, tc.srcs, filepath.Join(dstDir, tc.name+".tar.gz"))
			if err != nil && !errors.Is(err, tc.err) {
				t.Errorf("case %q: got unexpected error: %v", tc.name, err)
			}

			if tc.expWritten != written {
				t.Errorf("case %q: written bytes got %d want: %v", tc.name, written, tc.expWritten)
			}
		})
	}
}

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
		return 0, pr.CloseWithError(err)
	}

	<-done

	if err := ioutil.WriteFile(dst, content, 0644); err != nil {
		return 0, err
	}

	return written, nil
}
