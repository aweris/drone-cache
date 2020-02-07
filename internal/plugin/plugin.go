// Package plugin for caching directories using given backends
package plugin

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/meltwater/drone-cache/cache"
	"github.com/meltwater/drone-cache/cache/archive"
	"github.com/meltwater/drone-cache/cache/key"
	keygen "github.com/meltwater/drone-cache/cache/key/generator"
	"github.com/meltwater/drone-cache/internal/metadata"
	"github.com/meltwater/drone-cache/storage/backend"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type (
	// Config plugin-specific parameters and secrets.
	Config struct {
		ArchiveFormat    string
		Backend          string
		CacheKeyTemplate string

		CompressionLevel int

		Debug        bool
		SkipSymlinks bool
		Rebuild      bool
		Restore      bool

		Mount []string

		S3         backend.S3Config
		FileSystem backend.FileSystemConfig
		SFTP       backend.SFTPConfig
		Azure      backend.AzureConfig
		GCS        backend.GCSConfig
	}

	// Plugin stores metadata about current plugin.
	Plugin struct {
		logger log.Logger

		Metadata metadata.Metadata
		Config   Config
	}

	// Error recognized error from plugin.
	Error string
)

func (e Error) Error() string { return string(e) }

func (e Error) Unwrap() error { return e }

func New(logger log.Logger) *Plugin {
	return &Plugin{logger: logger}
}

// Exec entry point of Plugin, where the magic happens.
func (p *Plugin) Exec() error {
	cfg := p.Config

	// 1. Check parameters
	if cfg.Debug {
		level.Debug(p.logger).Log("msg", "DEBUG MODE enabled!")

		for _, pair := range os.Environ() {
			level.Debug(p.logger).Log("var", pair)
		}

		level.Debug(p.logger).Log("msg", "plugin initialized with config", "config", fmt.Sprintf("%#v", p.Config))
		level.Debug(p.logger).Log("msg", "plugin initialized with metadata", "metadata", fmt.Sprintf("%#v", p.Metadata))
	}

	if cfg.Rebuild && cfg.Restore {
		return errors.New("rebuild and restore are mutually exclusive, please set only one of them")
	}

	g := keygen.New(p.logger, p.Metadata)

	_, err := g.ParseTemplate(cfg.CacheKeyTemplate)
	if err != nil {
		return fmt.Errorf("parse, <%s> as cache key template, falling back to default %w", cfg.CacheKeyTemplate, err)
	}

	// 2. Initialize backend
	storage, err := storage.FromConfig(p.logger, cfg)
	if err != nil {
		return fmt.Errorf("initialize, <%s> as backend %w", cfg.Backend, err)
	}

	// 3. Initialize cache
	c := cache.New(p.logger, storage,
		archive.FromFormat(p.logger, cfg.ArchiveFormat,
			archive.WithSkipSymlinks(cfg.SkipSymlinks),
			archive.WithCompressionLevel(cfg.CompressionLevel),
		),
	)

	// 4. Select mode
	if cfg.Rebuild {
		if err := processRebuild(p.logger, c, g, p.Config.CacheKeyTemplate, p.Metadata, p.Config.Mount); err != nil {
			return Error(fmt.Sprintf("[IMPORTANT] build cache, process rebuild failed, %v\n", err))
		}
	}

	if cfg.Restore {
		if err := processRestore(p.logger, c, g, p.Config.CacheKeyTemplate, p.Metadata, p.Config.Mount); err != nil {
			return Error(fmt.Sprintf("[IMPORTANT] restore cache, process restore failed, %v\n", err))
		}
	}

	return nil
}

// processRebuild the remote cache from the local environment
func processRebuild(l log.Logger, c cache.Cache, g key.Generator, cacheKeyTmpl string, m metadata.Metadata, mountedDirs []string) error {
	now := time.Now()
	branch := m.Commit.Branch

	for _, mount := range mountedDirs {
		if _, err := os.Stat(mount); err != nil {
			return fmt.Errorf("mount <%s>, make sure file or directory exists and readable %w", mount, err)
		}

		key, err := g.Generate(cacheKeyTmpl, mount, branch)
		if err != nil {
			return fmt.Errorf("generate cache key %w", err)
		}

		path := filepath.Join(m.Repo.Name, key)

		level.Info(l).Log("msg", "rebuilding cache for directory", "local", mount, "remote", path)

		if err := c.Push(mount, path); err != nil {
			return fmt.Errorf("upload %w", err)
		}
	}

	level.Info(l).Log("msg", "cache built", "took", time.Since(now))

	return nil
}

// processRestore the local environment from the remote cache.
func processRestore(l log.Logger, c cache.Cache, g key.Generator, cacheKeyTmpl string, m metadata.Metadata, mountedDirs []string) error {
	now := time.Now()
	branch := m.Commit.Branch

	for _, mount := range mountedDirs {
		key, err := g.Generate(cacheKeyTmpl, mount, branch)
		if err != nil {
			return fmt.Errorf("generate cache key %w", err)
		}

		path := filepath.Join(m.Repo.Name, key)
		level.Info(l).Log("msg", "restoring directory", "local", mount, "remote", path)

		if err := c.Pull(path, mount); err != nil {
			return fmt.Errorf("download %w", err)
		}
	}

	level.Info(l).Log("msg", "cache restored", "took", time.Since(now))

	return nil
}
