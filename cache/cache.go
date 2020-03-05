// Package cache provides functionality for cache storage
package cache

import (
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/meltwater/drone-cache/archive"
	"github.com/meltwater/drone-cache/key"
	"github.com/meltwater/drone-cache/key/generator"
	"github.com/meltwater/drone-cache/storage"
)

// Default Key generator to fallback for the cache.
var defaultGen key.Generator = generator.NewStatic()

// Cache defines Cache functionality and stores configuration.
type Cache interface {
	Rebuilder
	Restorer
	Flusher
}

// cache TODO default cache!
type cache struct {
	logger log.Logger

	a  archive.Archive
	s  storage.Storage
	g  key.Generator
	fg key.Generator

	namespace string
}

// New creates a new cache with given parameters.
func New(logger log.Logger, s storage.Storage, a archive.Archive, g key.Generator, opts ...Option) Cache {
	options := options{}

	for _, o := range opts {
		o.apply(&options)
	}

	return &cache{
		logger:    log.With(logger, "component", "cache"),
		a:         a,
		s:         s,
		g:         g,
		namespace: options.namespace,
		fg:        options.fallbackGenerator,
	}
}

// TODO: write a note to redirect traffic to related files??
// or create struct and promote?

// Helpers

func (c *cache) generateKey(parts ...string) (string, error) {
	key, err := c.g.Generate(parts...)
	if err == nil {
		return key, nil
	}

	if c.fg != nil {
		level.Error(c.logger).Log("msg", "falling back to fallback key generator", "err", err)

		key, err = c.fg.Generate(parts...)
		if err == nil {
			return key, nil
		}
	}

	level.Error(c.logger).Log("msg", "falling back to default key generator", "err", err)

	key, err = defaultGen.Generate(parts...)
	if err != nil {
		return "", fmt.Errorf("generate key %w", err)
	}

	return key, nil
}
