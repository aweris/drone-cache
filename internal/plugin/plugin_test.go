package plugin

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/meltwater/drone-cache/archive"
	"github.com/meltwater/drone-cache/internal/metadata"
	"github.com/meltwater/drone-cache/storage/backend"
	"github.com/meltwater/drone-cache/storage/backend/filesystem"
	"github.com/meltwater/drone-cache/storage/backend/s3"

	"github.com/go-kit/kit/log"
	"github.com/minio/minio-go"
)

const (
	defaultEndpoint             = "127.0.0.1:9000"
	defaultAccessKey            = "AKIAIOSFODNN7EXAMPLE"
	defaultSecretAccessKey      = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	defaultRegion               = "eu-west-1"
	testStorageOperationTimeout = 5 * time.Second
	useSSL                      = false
)

var (
	endpoint        = getEnv("TEST_ENDPOINT", defaultEndpoint)
	accessKey       = getEnv("TEST_ACCESS_KEY", defaultAccessKey)
	secretAccessKey = getEnv("TEST_SECRET_KEY", defaultSecretAccessKey)
)

func TestRebuild(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-rebuild"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	fPath := filepath.Join(dirPath, "file_to_cache.txt")
	file, fErr := os.Create(fPath)
	if fErr != nil {
		t.Fatal(fErr)
	}

	content := make([]byte, 1024)
	rand.Read(content)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	absPath, err := filepath.Abs(fPath)
	if err != nil {
		t.Fatal(err)
	}

	linkAbsPath, err := filepath.Abs(filepath.Join(dirPath, "symlink_to_cache.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(absPath, linkAbsPath); err != nil {
		t.Fatal(err)
	}

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "", "tar")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin exec failed, error: %v\n", err)
	}
}

func TestRebuildSkipSymlinks(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-rebuild-skip-symlink"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	fPath := filepath.Join(dirPath, "file_to_cache.txt")
	file, fErr := os.Create(fPath)
	if fErr != nil {
		t.Fatal(fErr)
	}

	content := make([]byte, 1024)
	rand.Read(content)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	absPath, err := filepath.Abs(fPath)
	if err != nil {
		t.Fatal(err)
	}

	linkAbsPath, err := filepath.Abs(filepath.Join(dirPath, "symlink_to_cache.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(absPath, linkAbsPath); err != nil {
		t.Fatal(err)
	}

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "", "tar")
	plugin.Config.SkipSymlinks = true

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin exec failed, error: %v\n", err)
	}
}

func TestRebuildWithCacheKey(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-rebuild-cache-key"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	file, fErr := os.Create(filepath.Join(dirPath, "file_to_cache.txt"))
	if fErr != nil {
		t.Fatal(fErr)
	}

	content := make([]byte, 1024)
	rand.Read(content)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "{{ .Repo.Name }}_{{ .Commit.Branch }}_{{ .Build.Number }}", "tar")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin exec failed, error: %v\n", err)
	}
}

func TestRebuildWithGzip(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-rebuild-with-gzip"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	file, fErr := os.Create(filepath.Join(dirPath, "file_to_cache.txt"))
	if fErr != nil {
		t.Fatal(fErr)
	}

	content := make([]byte, 1024)
	rand.Read(content)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "", "gzip")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin exec failed, error: %v\n", err)
	}
}

func TestRebuildWithFilesystem(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-rebuild-filesystem"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	file, fErr := os.Create(filepath.Join(dirPath, "file_to_cache.txt"))
	if fErr != nil {
		t.Fatal(fErr)
	}

	content := make([]byte, 1024)
	rand.Read(content)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	plugin := newTestPlugin(name, backend.FileSystem, true, false, []string{dirPath}, "", "gzip")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin exec failed, error: %v\n", err)
	}
}

func TestRebuildNonExisting(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-rebuild-non-existing"
	_, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	plugin := newTestPlugin(name, backend.S3, true, false, []string{"./nonexisting/path"}, "", "tar")

	if err := plugin.Exec(); err == nil {
		t.Error("plugin exec did not fail as expected, error: <nil>")
	}
}

func TestRestore(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-restore"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll("./tmp/2", 0755); err != nil {
		t.Fatal(err)
	}

	fPath := filepath.Join(dirPath, "file_to_cache.txt")
	file, cErr := os.Create(fPath)
	if cErr != nil {
		t.Fatal(cErr)
	}

	content := make([]byte, 1024)
	rand.Read(content)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	file1, fErr1 := os.Create(filepath.Join(dirPath, "file1_to_cache.txt"))
	if fErr1 != nil {
		t.Fatal(fErr1)
	}

	content = make([]byte, 1024)
	rand.Read(content)
	if _, err := file1.Write(content); err != nil {
		t.Fatal(err)
	}

	file1.Sync()
	file1.Close()

	absPath, err := filepath.Abs(fPath)
	if err != nil {
		t.Fatal(err)
	}

	linkAbsPath, err := filepath.Abs(filepath.Join(dirPath, "symlink_to_cache.txt"))
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(absPath, linkAbsPath); err != nil {
		t.Fatal(err)
	}

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "", "tar")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (rebuild mode) exec failed, error: %v\n", err)
	}

	if err := os.RemoveAll("./tmp"); err != nil {
		t.Fatal(err)
	}

	plugin.Config.Rebuild = false
	plugin.Config.Restore = true
	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (restore mode) exec failed, error: %v\n", err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file1_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}

	target, err := os.Readlink(filepath.Join(dirPath, "symlink_to_cache.txt"))
	if err != nil {
		t.Error(err)
	}

	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Error(err)
	}
}

func TestRestoreWithCacheKey(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-restore-cache-key"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	file, cErr := os.Create(filepath.Join(dirPath, "file_to_cache.txt"))
	if cErr != nil {
		t.Fatal(cErr)
	}

	content := make([]byte, 1024)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	file1, fErr1 := os.Create(filepath.Join(dirPath, "file1_to_cache.txt"))
	if fErr1 != nil {
		t.Fatal(fErr1)
	}

	content = make([]byte, 1024)
	rand.Read(content)
	if _, err := file1.Write(content); err != nil {
		t.Fatal(err)
	}

	file1.Sync()
	file1.Close()

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "{{ .Repo.Name }}_{{ .Commit.Branch }}_{{ .Build.Number }}", "tar")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (rebuild mode) exec failed, error: %v\n", err)
	}

	if err := os.RemoveAll("./tmp"); err != nil {
		t.Fatal(err)
	}

	plugin.Config.Rebuild = false
	plugin.Config.Restore = true
	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (restore mode) exec failed, error: %v\n", err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file1_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}
}

func TestRestoreWithGzip(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-restore-with-gzip"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	file, cErr := os.Create(filepath.Join(dirPath, "file_to_cache.txt"))
	if cErr != nil {
		t.Fatal(cErr)
	}

	content := make([]byte, 1024)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	file1, fErr1 := os.Create(filepath.Join(dirPath, "file1_to_cache.txt"))
	if fErr1 != nil {
		t.Fatal(fErr1)
	}

	content = make([]byte, 1024)
	rand.Read(content)
	if _, err := file1.Write(content); err != nil {
		t.Fatal(err)
	}

	file1.Sync()
	file1.Close()

	plugin := newTestPlugin(name, backend.S3, true, false, []string{dirPath}, "", "gzip")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (rebuild mode) exec failed, error: %v\n", err)
	}

	if err := os.RemoveAll("./tmp"); err != nil {
		t.Fatal(err)
	}

	plugin.Config.Rebuild = false
	plugin.Config.Restore = true
	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (restore mode) exec failed, error: %v\n", err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file1_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}
}

func TestRestoreWithFilesystem(t *testing.T) {
	t.Parallel()

	name := "drone-cache-test-restore-with-filesystem"
	tmpDir, cleanUp := setup(t, name)
	t.Cleanup(cleanUp)

	dirPath := filepath.Join(tmpDir, "1")
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		t.Fatal(err)
	}

	file, cErr := os.Create(filepath.Join(dirPath, "file_to_cache.txt"))
	if cErr != nil {
		t.Fatal(cErr)
	}

	content := make([]byte, 1024)
	if _, err := file.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if mkErr1 := os.MkdirAll(dirPath, 0755); mkErr1 != nil {
		t.Fatal(mkErr1)
	}

	file1, fErr1 := os.Create(filepath.Join(dirPath, "file1_to_cache.txt"))
	if fErr1 != nil {
		t.Fatal(fErr1)
	}

	content = make([]byte, 1024)
	rand.Read(content)
	if _, err := file1.Write(content); err != nil {
		t.Fatal(err)
	}

	file1.Sync()
	file1.Close()

	plugin := newTestPlugin(name, backend.FileSystem, true, false, []string{dirPath}, "", "gzip")

	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (rebuild mode) exec failed, error: %v\n", err)
	}

	if err := os.RemoveAll("./tmp"); err != nil {
		t.Fatal(err)
	}

	plugin.Config.Rebuild = false
	plugin.Config.Restore = true
	if err := plugin.Exec(); err != nil {
		t.Errorf("plugin (restore mode) exec failed, error: %v\n", err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}

	if _, err := os.Stat(filepath.Join(dirPath, "file1_to_cache.txt")); os.IsNotExist(err) {
		t.Error(err)
	}
}

// Helpers

func newTestPlugin(bucket, backend string, rebuild, restore bool, mount []string, cacheKey, archiveFmt string) Plugin {
	var logger log.Logger
	if testing.Verbose() {
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	} else {
		logger = log.NewNopLogger()
	}

	return Plugin{
		logger: logger,
		Metadata: metadata.Metadata{
			Repo: metadata.Repo{
				Branch: "master",
				Name:   "drone-cache",
			},
			Commit: metadata.Commit{
				Branch: "master",
			},
		},
		Config: Config{
			ArchiveFormat:           archiveFmt,
			CompressionLevel:        archive.DefaultCompressionLevel,
			Backend:                 backend,
			CacheKeyTemplate:        cacheKey,
			Mount:                   mount,
			Rebuild:                 rebuild,
			Restore:                 restore,
			StorageOperationTimeout: testStorageOperationTimeout,

			FileSystem: filesystem.Config{
				CacheRoot: "../../tmp/testdata/cache",
			},

			S3: s3.Config{
				ACL:        "private",
				Bucket:     bucket,
				Encryption: "",
				Endpoint:   endpoint,
				Key:        accessKey,
				PathStyle:  true, // Should be true for minio and false for AWS.
				Region:     defaultRegion,
				Secret:     secretAccessKey,
			},
		},
	}
}

func newMinioClient() (*minio.Client, error) {
	minioClient, err := minio.New(endpoint, accessKey, secretAccessKey, useSSL)
	if err != nil {
		return nil, err
	}
	return minioClient, nil
}

func setup(t *testing.T, name string) (string, func()) {
	t.Log("Setting up")
	// Notice: There's a lot of room to improve here:
	// - Tear down whole minio rather than cleaning each time!

	minioClient, err := newMinioClient()
	if err != nil {
		t.Fatalf("unexpectedly failed creating minioclient %v", err)
	}

	t.Log("Creating bucket")
	if err = minioClient.MakeBucket(name, defaultRegion); err != nil {
		t.Fatalf("unexpectedly failed creating bucket <%s> %v", name, err)
	}

	t.Log("Creating directory")
	tmpDir, err := ioutil.TempDir("", name+"-testdir-*")
	if err != nil {
		t.Fatalf("unexpectedly failed creating the temp dir: %v", err)
	}

	t.Log("Setup completed!")
	return tmpDir, func() {
		t.Log("Cleaning up")
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Fatalf("unexpectedly failed remove tmp dir <%s> %v", name, err)
			t.Fatal(err)
		}
		t.Log("Removing all objects...")
		if err = removeAllObjects(minioClient, name); err != nil {
			t.Fatal(err)
		}

		t.Log("Removing bucket...")
		if err = minioClient.RemoveBucket(name); err != nil {
			t.Fatal(err)
		}
	}
}

func removeAllObjects(minioClient *minio.Client, bucket string) error {
	objects := make(chan string)
	errors := make(chan error)

	go func() {
		defer close(objects)
		defer close(errors)

		for object := range minioClient.ListObjects(bucket, "", true, nil) {
			if object.Err != nil {
				errors <- object.Err
			}
			objects <- object.Key
		}
	}()

	for {
		select {
		case object, open := <-objects:
			if !open {
				return nil
			}
			if err := minioClient.RemoveObject(bucket, object); err != nil {
				return fmt.Errorf("remove all objects failed, %v", err)
			}
		case err, open := <-errors:
			if !open {
				return nil
			}
			if err != nil {
				return fmt.Errorf("remove all objects failed, while fetching %v", err)
			}

			return nil
		}
	}
}

func getEnv(key, defaultVal string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return value
}
