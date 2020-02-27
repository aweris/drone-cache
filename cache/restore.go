package cache

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-kit/kit/log/level"
)

// Restorer TODO
type Restorer interface {
	// Restore TODO
	Restore(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error
}

// Restore TODO
func (c *cache) Restore(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error {
	level.Info(c.logger).Log("msg", "restoring  cache")
	now := time.Now()

	key, err := c.generateKey(keyTempl)
	if err != nil {
		fmt.Errorf("generate key %w", err)
	}

	var wg sync.WaitGroup

	errs := make(chan error, len(srcs))
	defer close(errs)

	for _, src := range srcs {
		dst := filepath.Join(c.namespace, key, src)

		level.Info(c.logger).Log("msg", "restoring directory", "local", src, "remote", dst)

		wg.Add(1)
		go func(dst, src string) {
			defer wg.Done()

			if err := c.restore(dst, src); err != nil {
				errs <- fmt.Errorf("download from <%s> to <%s> %w", dst, src, err)
			}
		}(dst, src)
	}

	wg.Wait()

	if err := <-errs; err != nil {
		return fmt.Errorf("restore failed %w", err)
	}

	level.Info(c.logger).Log("msg", "cache restored", "took", time.Since(now))

	return nil
}

// restore fetches the archived file from the cache and restores to the host machine's file system.
func (c *cache) restore(dst, src string) error {
	pr, pw := io.Pipe()
	defer pr.Close()

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer pw.Close()
		defer close(done)

		level.Info(c.logger).Log("msg", "downloading archived directory", "remote", dst, "local", src)

		if err := c.s.Get(dst, pw); err != nil { // TODO: do we have close anything?
			pw.CloseWithError(fmt.Errorf("get file from storage backend, pipe writer failed %w", err))
		}
	}()

	level.Info(c.logger).Log("msg", "extracting archived directory", "remote", dst, "local", src)

	// TODO: Make sure not exit before writer finishes!!
	written, err := c.a.Extract(src, pr)
	if err != nil {
		return pr.CloseWithError(fmt.Errorf("extract files from downloaded archive, pipe reader failed %w", err))
	}

	<-done

	level.Debug(c.logger).Log(
		"msg", "archive extracted",
		"local", src,
		"remote", dst,
		"raw size", written, // TODO: Add ratio!!
	)

	return nil
}
