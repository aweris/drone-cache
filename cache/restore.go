package cache

// Restorer TODO
type Restorer interface {

	// Restore TODO
	Restore(srcs []string, keyTempl string, fallbackKeyTmpls ...string) error
}
