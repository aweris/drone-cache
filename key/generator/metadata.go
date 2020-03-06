package generator

import (
	// #nosec
	"errors"
	"fmt"
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

// Metadata TODO: maybe not exported?
type Metadata struct {
	logger log.Logger

	data    metadata.Metadata
	funcMap template.FuncMap
}

// NewMetadata creates a new Key Generator.
func NewMetadata(logger log.Logger, data metadata.Metadata) *Metadata {
	return &Metadata{
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
func (g *Metadata) Generate(tmpls ...string) (string, error) {
	// NOTICE: for now only consume a single template which will be changed.
	tmpl := tmpls[0]

	key, err := g.generateFromTemplate(tmpl)
	if err != nil {
		return "", fmt.Errorf("metadata key generator %w", err)
	}

	return key, nil
}

func (g *Metadata) generateFromTemplate(tmpl string) (string, error) {
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

	return b.String(), nil
}

// // Generate generates key from given template as parameter or fallbacks hash.
// func (g *Metadata) Generate(tmpls ...string) (string, error) {
// 	// NOTICE: for now only consume a single template which will be changed.
// 	tmpl := tmpls[0]
// 	key, err := g.generateFromTemplate(tmpl, path)
// 	if err != nil {
// 		level.Error(g.logger).Log("msg", "falling back to default key", "err", err)

// 		key, err = hash(append([]string{path}, fallback...)...)
// 		if err != nil {
// 			return "", fmt.Errorf("generate hash key for mounted %w", err)
// 		}
// 	}

// 	return key, nil
// }

// func (g *Metadata) generateFromTemplate(tmpl string, path string) (string, error) {
// 	level.Info(g.logger).Log("msg", "using provided cache key template")

// 	if tmpl == "" {
// 		return "", errors.New("cache key template is empty")
// 	}

// 	t, err := g.ParseTemplate(tmpl)
// 	if err != nil {
// 		return "", fmt.Errorf("parse, <%s> as cache key template, falling back to default %w", tmpl, err)
// 	}

// 	var b strings.Builder

// 	err = t.Execute(&b, g.data)
// 	if err != nil {
// 		return "", fmt.Errorf("build, <%s> as cache key, falling back to default %w", tmpl, err)
// 	}

// 	return filepath.Join(b.String(), path), nil
// }

// ParseTemplate parses template.
func (g *Metadata) ParseTemplate(tmpl string) (*template.Template, error) {
	return template.New("cacheKey").Funcs(g.funcMap).Parse(tmpl)
}

// Helpers

// checksumFunc TODO
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
