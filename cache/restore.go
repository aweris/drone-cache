package cache

import (
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/meltwater/drone-cache/archive"
	"github.com/meltwater/drone-cache/key"
	"github.com/meltwater/drone-cache/storage"
)

type restorer struct {
	logger log.Logger

	a  archive.Archive
	s  storage.Storage
	g  key.Generator
	fg key.Generator

	namespace string
}

func newRestorer(logger log.Logger, s storage.Storage, a archive.Archive, g key.Generator, fg key.Generator, namespace string) restorer {
	return restorer{logger, a, s, g, fg, namespace}
}

// Restore TODO
func (r restorer) Restore(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error {
	level.Info(r.logger).Log("msg", "restoring  cache")

	now := time.Now()

	key, err := r.generateKey(keyTempl)
	if err != nil {
		return fmt.Errorf("generate key %w", err)
	}

	var (
		wg   sync.WaitGroup
		errs = &MultiError{}
	)

	for _, src := range srcs {
		dst := filepath.Join(r.namespace, key, src)

		level.Info(r.logger).Log("msg", "restoring directory", "local", src, "remote", dst)

		wg.Add(1) //nolint:gomnd

		go func(dst, src string) {
			defer wg.Done()

			if err := r.restore(dst, src); err != nil {
				errs.Add(fmt.Errorf("download from <%s> to <%s> %w", dst, src, err))
			}
		}(dst, src)
	}

	wg.Wait()

	if errs.Err() != nil {
		return fmt.Errorf("restore failed %w", err)
	}

	level.Info(r.logger).Log("msg", "cache restored", "took", time.Since(now))

	return nil
}

// restore fetches the archived file from the cache and restores to the host machine's file system.
func (r restorer) restore(dst, src string) error {
	pr, pw := io.Pipe()
	defer pr.Close()

	go func() {
		defer pw.Close()

		level.Info(r.logger).Log("msg", "downloading archived directory", "remote", dst, "local", src)

		if err := r.s.Get(dst, pw); err != nil {
			if err := pw.CloseWithError(fmt.Errorf("get file from storage backend, pipe writer failed %w", err)); err != nil {
				level.Error(r.logger).Log("msg", "pw close", "err", err)
			}
		}
	}()

	level.Info(r.logger).Log("msg", "extracting archived directory", "remote", dst, "local", src)

	written, err := r.a.Extract(src, pr)
	if err != nil {
		err = fmt.Errorf("extract files from downloaded archive, pipe reader failed %w", err)
		if err := pr.CloseWithError(err); err != nil {
			level.Error(r.logger).Log("msg", "pr close", "err", err)
		}

		return err
	}

	level.Debug(r.logger).Log(
		"msg", "archive extracted",
		"local", src,
		"remote", dst,
		"raw size", written,
	)

	return nil
}

// Helpers

func (r restorer) generateKey(parts ...string) (string, error) {
	key, err := r.g.Generate(parts...)
	if err == nil {
		return key, nil
	}

	if r.fg != nil {
		level.Error(r.logger).Log("msg", "falling back to fallback key generator", "err", err)

		key, err = r.fg.Generate(parts...)
		if err == nil {
			return key, nil
		}
	}

	level.Error(r.logger).Log("msg", "falling back to default key generator", "err", err)

	key, err = defaultGen.Generate(parts...)
	if err != nil {
		return "", fmt.Errorf("generate key %w", err)
	}

	return key, nil
}
