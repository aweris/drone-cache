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
		return fmt.Errorf("generate key %w", err)
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

	go func() {
		defer pw.Close()

		level.Info(c.logger).Log("msg", "downloading archived directory", "remote", dst, "local", src)

		if err := c.s.Get(dst, pw); err != nil { // TODO: do we have close anything?
			if err := pw.CloseWithError(fmt.Errorf("get file from storage backend, pipe writer failed %w", err)); err != nil {
				level.Error(c.logger).Log("msg", "pw close", "err", err)
			}
		}
	}()

	level.Info(c.logger).Log("msg", "extracting archived directory", "remote", dst, "local", src)

	// TODO: Make sure not exit before writer finishes!!
	written, err := c.a.Extract(src, pr)
	if err != nil {
		err = fmt.Errorf("extract files from downloaded archive, pipe reader failed %w", err)
		// TODO: Introduce runutils ? ioutils to close and log error
		if err := pr.CloseWithError(err); err != nil {
			level.Error(c.logger).Log("msg", "pr close", "err", err)
		}

		return err
	}

	level.Debug(c.logger).Log(
		"msg", "archive extracted",
		"local", src,
		"remote", dst,
		"raw size", written, // TODO: Add ratio!!
	)

	return nil
}
