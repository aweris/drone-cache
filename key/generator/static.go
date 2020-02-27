package generator

import "path/filepath"

type staticGenerator struct{}

func NewStatic() *staticGenerator {
	return &staticGenerator{}
}

func (s *staticGenerator) Generate(parts ...string) (string, error) {
	return filepath.Join(parts...), nil
}
