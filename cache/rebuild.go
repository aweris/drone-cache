package cache

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-kit/kit/log/level"
)

// Rebuilder TODO
type Rebuilder interface {
	// Rebuild TODO
	Rebuild(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error
}

// Rebuild TODO
func (c *cache) Rebuild(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error {
	level.Info(c.logger).Log("msg", "rebuilding cache")

	now := time.Now()

	key, err := c.generateKey(keyTempl)
	if err != nil {
		return fmt.Errorf("generate key %w", err)
	}

	var wg sync.WaitGroup

	errs := make(chan error, len(srcs))

	for _, src := range srcs {
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("source <%s>, make sure file or directory exists and readable %w", src, err)
		}

		dst := filepath.Join(c.namespace, key, src)

		level.Info(c.logger).Log("msg", "rebuilding cache for directory", "local", src, "remote", dst)

		wg.Add(1)

		go func(dst, src string) {
			defer wg.Done()

			if err := c.rebuild(src, dst); err != nil {
				errs <- fmt.Errorf("upload from <%s> to <%s> %w", src, dst, err)
			}
		}(dst, src)
	}

	wg.Wait()
	close(errs)

	if err := <-errs; err != nil {
		return fmt.Errorf("rebuild failed %w", err)
	}

	level.Info(c.logger).Log("msg", "cache built", "took", time.Since(now))

	return nil
}

// rebuild pushes the archived file to the cache.
func (c *cache) rebuild(src, dst string) error {
	src, err := filepath.Abs(filepath.Clean(src))
	if err != nil {
		return fmt.Errorf("read source directory %w", err)
	}

	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		defer pw.Close()

		level.Info(c.logger).Log("msg", "archiving directory", "src", src)

		written, err := c.a.Create([]string{src}, pw)
		if err != nil {
			if err := pw.CloseWithError(fmt.Errorf("archive write, pipe writer failed %w", err)); err != nil {
				level.Error(c.logger).Log("msg", "pw close", "err", err)
			}
		}

		// TODO: Calculate stats!
		level.Debug(c.logger).Log(
			"msg", "archive created",
			"local", src,
			"remote", dst,
			"raw size", written,
			// "compressed size", stat.Size(),
			// "compression ratio %", written/stat.Size()*100,
		)
	}()

	level.Info(c.logger).Log("msg", "uploading archived directory", "local", src, "remote", dst)

	// WriteCloser? Make sure not exit before writer finishes!!
	if err := c.s.Put(dst, pr); err != nil {
		err = fmt.Errorf("upload file <%s>, pipe reader failed %w", src, err)
		// TODO: Introduce runutils ? ioutils to close and log error
		if err := pr.CloseWithError(err); err != nil {
			level.Error(c.logger).Log("msg", "pr close", "err", err)
		}

		return err
	}

	return nil
}
