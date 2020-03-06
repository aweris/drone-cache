// Package plugin for caching directories using given backends
package plugin

import (
	"errors"
	"fmt"
	"os"

	"github.com/meltwater/drone-cache/archive"
	"github.com/meltwater/drone-cache/cache"
	"github.com/meltwater/drone-cache/internal/metadata"
	keygen "github.com/meltwater/drone-cache/key/generator"
	"github.com/meltwater/drone-cache/storage"
	"github.com/meltwater/drone-cache/storage/backend"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Error recognized error from plugin.
type Error string

// Error TODO
func (e Error) Error() string { return string(e) }

// Unwrap TODO
func (e Error) Unwrap() error { return e }

// Plugin stores metadata about current plugin.
type Plugin struct {
	logger log.Logger

	Metadata metadata.Metadata
	Config   Config
}

// New TODO
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

	// FLUSH

	if cfg.Rebuild && cfg.Restore {
		return errors.New("rebuild and restore are mutually exclusive, please set only one of them")
	}

	generator := keygen.NewMetadata(p.logger, p.Metadata)

	_, err := generator.ParseTemplate(cfg.CacheKeyTemplate)
	if err != nil {
		return fmt.Errorf("parse, <%s> as cache key template, falling back to default %w", cfg.CacheKeyTemplate, err)
	}

	// 2. Initialize storage backend.
	b, err := backend.FromConfig(p.logger, cfg.Backend, backend.Config{
		Debug:      cfg.Debug,
		Azure:      cfg.Azure,
		FileSystem: cfg.FileSystem,
		GCS:        cfg.GCS,
		S3:         cfg.S3,
		SFTP:       cfg.SFTP,
	})
	if err != nil {
		return fmt.Errorf("initialize backend <%s> %w", cfg.Backend, err)
	}

	// 3. Initialize cache.
	c := cache.New(p.logger,
		storage.New(p.logger, b, cfg.StorageOperationTimeout),
		archive.FromFormat(p.logger, cfg.ArchiveFormat,
			archive.WithSkipSymlinks(cfg.SkipSymlinks),
			archive.WithCompressionLevel(cfg.CompressionLevel),
		),
		generator,
		// Missing Documentation
		cache.WithNamespace(p.Metadata.Repo.Name),
		// Missing Documentation
		cache.WithFallbackGenerator(keygen.NewHash(p.Metadata.Commit.Branch)),
	)

	// 4. Select mode
	if cfg.Rebuild {
		if err := c.Rebuild(p.Config.Mount, p.Config.CacheKeyTemplate); err != nil {
			level.Debug(p.logger).Log("err", fmt.Sprintf("%+v\n", err))
			return Error(fmt.Sprintf("[IMPORTANT] build cache, process rebuild failed, %v\n", err))
		}
	}

	if cfg.Restore {
		if err := c.Restore(p.Config.Mount, p.Config.CacheKeyTemplate); err != nil {
			level.Debug(p.logger).Log("err", fmt.Sprintf("%+v\n", err))
			return Error(fmt.Sprintf("[IMPORTANT] restore cache, process restore failed, %v\n", err))
		}
	}

	// FLUSH

	return nil
}
