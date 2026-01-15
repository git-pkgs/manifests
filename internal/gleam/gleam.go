package gleam

import (
	"github.com/git-pkgs/manifests/internal/core"
	"github.com/BurntSushi/toml"
)

func init() {
	core.Register("hex", core.Manifest, &gleamTomlParser{}, core.ExactMatch("gleam.toml"))
}

// gleamTomlParser parses gleam.toml files.
type gleamTomlParser struct{}

type gleamToml struct {
	Dependencies    map[string]string `toml:"dependencies"`
	DevDependencies map[string]string `toml:"dev-dependencies"`
}

func (p *gleamTomlParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var gleam gleamToml
	if err := toml.Unmarshal(content, &gleam); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, version := range gleam.Dependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, version := range gleam.DevDependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return deps, nil
}
