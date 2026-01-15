package conda

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("conda", core.Manifest, &condaEnvParser{}, core.ExactMatch("environment.yml"))
	core.Register("conda", core.Manifest, &condaEnvParser{}, core.ExactMatch("environment.yaml"))
}

// condaEnvParser parses Conda environment.yml files.
type condaEnvParser struct{}

type condaEnvironment struct {
	Name         string `yaml:"name"`
	Channels     []string `yaml:"channels"`
	Dependencies []any    `yaml:"dependencies"`
}

func (p *condaEnvParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var env condaEnvironment
	if err := yaml.Unmarshal(content, &env); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, dep := range env.Dependencies {
		switch d := dep.(type) {
		case string:
			// Regular conda dependency: "name", "name=version", or "name=version=build"
			name, version := parseCondaSpec(d)
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		case map[string]any:
			// Skip nested pip dependencies - they belong to pypi ecosystem
		}
	}

	return deps, nil
}

// parseCondaSpec parses a Conda dependency spec like "name", "name=version", or "name=version=build".
func parseCondaSpec(spec string) (name, version string) {
	parts := strings.Split(spec, "=")
	name = parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}
	return name, version
}
