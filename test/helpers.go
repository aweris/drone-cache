package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/mholt/archiver"
)

// CreateTempFile TODO
func CreateTempFile(t testing.TB, name string, content []byte) (string, func()) {
	// t.Helper()

	tmpfile, err := ioutil.TempFile("", name+".*.testfile")
	if err != nil {
		t.Fatalf("unexpectedly failed creating the temp file: %v", err)
	}

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("unexpectedly failed writing to the temp file: %v", err)
	}

	if err := tmpfile.Close(); err != nil {
		t.Fatalf("unexpectedly failed closing the temp file: %v", err)
	}

	return tmpfile.Name(), func() { os.Remove(tmpfile.Name()) }
}

// CreateTempFileInDir TODO
func CreateTempFilesInDir(t testing.TB, name string, content []byte) (string, func()) {
	// t.Helper()

	tmpDir, err := ioutil.TempDir("", name+"-testdir-*")
	if err != nil {
		t.Fatalf("unexpectedly failed creating the temp dir: %v", err)
	}

	for i := 0; i < 3; i++ {
		tmpfile, err := ioutil.TempFile(tmpDir, name+".*.testfile")
		if err != nil {
			t.Fatalf("unexpectedly failed creating the temp file: %v", err)
		}

		if _, err := tmpfile.Write(content); err != nil {
			t.Fatalf("unexpectedly failed writing to the temp file: %v", err)
		}

		if err := tmpfile.Close(); err != nil {
			t.Fatalf("unexpectedly failed closing the temp file: %v", err)
		}
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

// CreateTempDir TODO
func CreateTempDir(t testing.TB, name string) (string, func()) {
	// t.Helper()

	tmpDir, err := ioutil.TempDir("", name+"-testdir-*")
	if err != nil {
		t.Fatalf("unexpectedly failed creating the temp dir: %v", err)
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

// IsDir TODO
func IsDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}

	return fi.IsDir()
}

// Expand TODO
func Expand(src string) ([]string, error) {
	paths := []string{}
	if err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk %q: %v\n", path, err)
		}

		if info.IsDir() {
			return nil
		}

		paths = append(paths, path)
		return nil
	}); err != nil {
		return nil, fmt.Errorf("walking the path %q: %v\n", src, err)
	}

	return paths, nil
}

// CreateArchive TODO
func CreateArchive(srcs []string, dst string) error {
	return archiver.Archive(srcs, dst)
}

// ExtractArchive TODO
func ExtractArchive(src string, dst string) error {
	return archiver.Unarchive(src, dst)
}
