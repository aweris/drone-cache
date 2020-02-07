package key

// Generator defines a cache key generator.
type Generator interface {
	// Generate generates key from given template as parameter or fallbacks hash
	Generate(tmpl, path string, fallback ...string) (string, error)
}
