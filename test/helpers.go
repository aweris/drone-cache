package test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/mholt/archiver"
)

// CreateTempFile TODO
func CreateTempFile(t testing.TB, name string, content []byte) (string, func()) {
	t.Helper()

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

// CreateTempFilesInDir TODO
func CreateTempFilesInDir(t testing.TB, name string, content []byte) (string, func()) {
	t.Helper()

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
	t.Helper()

	tmpDir, err := ioutil.TempDir("", name+"-testdir-*")
	if err != nil {
		t.Fatalf("unexpectedly failed creating the temp dir: %v", err)
	}

	return tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}

// CreateArchive TODO
func CreateArchive(srcs []string, dst string) error {
	return archiver.Archive(srcs, dst)
}

// ExtractArchive TODO
func ExtractArchive(src string, dst string) error {
	return archiver.Unarchive(src, dst)
}
