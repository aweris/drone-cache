package generator

import (
	"crypto/md5" // #nosec
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/meltwater/drone-cache/internal/metadata"
)

type MetadataGenerator struct {
	logger  log.Logger
	data    metadata.Metadata
	funcMap template.FuncMap
}

// New creates a new Key Generator.
func New(logger log.Logger, data metadata.Metadata) *MetadataGenerator {
	return &MetadataGenerator{
		logger: logger,
		data:   data,
		funcMap: template.FuncMap{
			"checksum": checksumFunc(logger),
			"epoch":    func() string { return strconv.FormatInt(time.Now().Unix(), 10) },
			"arch":     func() string { return runtime.GOARCH },
			"os":       func() string { return runtime.GOOS },
		},
	}
}

// Generate generates key from given template as parameter or fallbacks hash.
func (g *MetadataGenerator) Generate(tmpl string, path string, fallback ...string) (string, error) {
	key, err := g.generateFromTemplate(tmpl, path)
	if err != nil {
		level.Error(g.logger).Log("msg", "falling back to default key", "err", err)

		key, err = hash(append([]string{path}, fallback...)...)
		if err != nil {
			return "", fmt.Errorf("generate hash key for mounted %w", err)
		}
	}

	return key, nil
}

func (g *MetadataGenerator) generateFromTemplate(tmpl string, path string) (string, error) {
	level.Info(g.logger).Log("msg", "using provided cache key template")

	if tmpl == "" {
		return "", errors.New("cache key template is empty")
	}

	t, err := g.ParseTemplate(tmpl)
	if err != nil {
		return "", fmt.Errorf("parse, <%s> as cache key template, falling back to default %w", tmpl, err)
	}

	var b strings.Builder

	err = t.Execute(&b, g.data)
	if err != nil {
		return "", fmt.Errorf("build, <%s> as cache key, falling back to default %w", tmpl, err)
	}

	return filepath.Join(b.String(), path), nil
}

// ParseTemplate parses template.
func (g *MetadataGenerator) ParseTemplate(tmpl string) (*template.Template, error) {
	return template.New("cacheKey").Funcs(g.funcMap).Parse(tmpl)
}

// Helpers

// hash generates a key based on given strings (ie. filename paths and branch).
func hash(parts ...string) (string, error) {
	readers := make([]io.Reader, len(parts))
	for i, p := range parts {
		readers[i] = strings.NewReader(p)
	}

	return readerHasher(readers...)
}

// readerHasher generic md5 hash generater from io.Reader.
func readerHasher(readers ...io.Reader) (string, error) {
	h := md5.New() // #nosec

	for _, r := range readers {
		if _, err := io.Copy(h, r); err != nil {
			return "", fmt.Errorf("write reader as hash %w", err)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func checksumFunc(logger log.Logger) func(path string) string {
	return func(path string) string {
		absPath, err := filepath.Abs(filepath.Clean(path))
		if err != nil {
			level.Error(logger).Log("cache key template/checksum could not find file")
			return ""
		}

		f, err := os.Open(absPath)
		if err != nil {
			level.Error(logger).Log("cache key template/checksum could not open file")
			return ""
		}

		defer f.Close()

		str, err := readerHasher(f)
		if err != nil {
			level.Error(logger).Log("cache key template/checksum could not generate hash")
			return ""
		}

		return str
	}
}
