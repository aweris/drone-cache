package generator

import (
	"testing"
	"text/template"

	"github.com/go-kit/kit/log"
	"github.com/meltwater/drone-cache/internal/metadata"
)

func TestGenerate(t *testing.T) {
	l := log.NewNopLogger()
	g := MetadataGenerator{
		logger: l,
		data:   metadata.Metadata{Repo: metadata.Repo{Name: "RepoName"}},
		funcMap: template.FuncMap{
			"checksum": checksumFunc(l),
			"epoch":    func() string { return "1550563151" },
			"arch":     func() string { return "amd64" },
			"os":       func() string { return "darwin" },
		},
	}

	table := []struct {
		given    string
		expected string
	}{
		{`{{ .Repo.Name }}`, "RepoName"},
		{`{{ checksum "checksum_file_test.txt"}}`, "04a29c732ecbce101c1be44c948a50c6"},
		{`{{ checksum "../../docs/drone_env_vars.md"}}`, "f8b5b7f96f3ffaa828e4890aab290e59"},
		{`{{ epoch }}`, "1550563151"},
		{`{{ arch }}`, "amd64"},
		{`{{ os }}`, "darwin"},
	}

	for _, tt := range table {
		t.Run(tt.given, func(t *testing.T) {
			actual, err := g.Generate(tt.given, "")
			if err != nil {
				t.Errorf("generate failed, error: %v\n", err)
			}

			if actual != tt.expected {
				t.Errorf("generate failed, got: %s, want: %s\n", actual, tt.expected)
			}
		})
	}
}

func TestParseTemplate(t *testing.T) {
	l := log.NewNopLogger()
	g := MetadataGenerator{
		logger: l,
		data:   metadata.Metadata{Repo: metadata.Repo{Name: "RepoName"}},
		funcMap: template.FuncMap{
			"checksum": checksumFunc(l),
			"epoch":    func() string { return "1550563151" },
			"arch":     func() string { return "amd64" },
			"os":       func() string { return "darwin" },
		},
	}

	table := []struct {
		given string
	}{
		{`{{ .Repo.Name }}`},
		{`{{ checksum "checksum_file_test.txt"}}`},
		{`{{ epoch }}`},
		{`{{ arch }}`},
		{`{{ os }}`},
	}
	for _, tt := range table {
		t.Run(tt.given, func(t *testing.T) {
			_, err := g.ParseTemplate(tt.given)
			if err != nil {
				t.Errorf("parser template failed, error: %v\n", err)
			}
		})
	}
}

func TestHash(t *testing.T) {
	actual, err := hash("hash")
	if err != nil {
		t.Errorf("hash failed, error: %v\n", err)
	}

	expected := "0800fc577294c34e0b28ad2839435945"
	if actual != expected {
		t.Errorf("hash failed, got: %s, want: %s\n", actual, expected)
	}
}
