package generator

import "path/filepath"

type Static struct{}

func NewStatic() *Static {
	return &Static{}
}

func (s *Static) Generate(parts ...string) (string, error) {
	return filepath.Join(parts...), nil
}
