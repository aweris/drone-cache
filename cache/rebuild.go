package cache

// Rebuilder TODO
type Rebuilder interface {
	// Rebuild TODO
	Rebuild(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error
}
