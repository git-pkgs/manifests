package gleam

import (
	"net/url"

	"github.com/BurntSushi/toml"
	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("hex", core.Manifest, &gleamTomlParser{}, core.ExactMatch("gleam.toml"))
}

// gleamTomlParser parses gleam.toml files.
type gleamTomlParser struct{}

type gleamToml struct {
	Name            string         `toml:"name"`
	Version         string         `toml:"version"`
	Dependencies    map[string]any `toml:"dependencies"`
	DevDependencies map[string]any `toml:"dev-dependencies"`
}

func (p *gleamTomlParser) Parse(filename string, content []byte) (*core.Result, error) {
	var gleam gleamToml
	if err := toml.Unmarshal(content, &gleam); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration

	for name, value := range gleam.Dependencies {
		version, ok := value.(string)
		if !ok {
			continue
		}
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
		declarations = append(declarations, core.Declaration{
			Name:     name,
			Version:  version,
			Scope:    core.Runtime,
			Direct:   true,
			Location: "dependencies/" + url.PathEscape(name),
		})
	}

	for name, value := range gleam.DevDependencies {
		version, ok := value.(string)
		if !ok {
			continue
		}
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
		declarations = append(declarations, core.Declaration{
			Name:     name,
			Version:  version,
			Scope:    core.Development,
			Direct:   true,
			Location: "dev-dependencies/" + url.PathEscape(name),
		})
	}

	return &core.Result{
		Name:         gleam.Name,
		Version:      gleam.Version,
		Dependencies: deps,
		Declarations: declarations,
	}, nil
}
