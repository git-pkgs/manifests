package elm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
)

func init() {
	core.Register("elm", core.Manifest, &elmJSONParser{}, core.ExactMatch("elm.json"))
	core.Register("elm", core.Manifest, &elmPackageJSONParser{}, core.ExactMatch("elm-package.json"))
}

// elmJSONParser parses elm.json files (Elm 0.19+).
type elmJSONParser struct{}

type elmJSON struct {
	Dependencies     elmDependencies `json:"dependencies"`
	TestDependencies elmDependencies `json:"test-dependencies"`
}

type elmDependencies struct {
	Direct   map[string]string `json:"direct"`
	Indirect map[string]string `json:"indirect"`
}

func (p *elmJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var elm elmJSON
	if err := json.Unmarshal(content, &elm); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	// Direct dependencies
	for name, version := range elm.Dependencies.Direct {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	// Indirect dependencies
	for name, version := range elm.Dependencies.Indirect {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	// Test dependencies (direct)
	for name, version := range elm.TestDependencies.Direct {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Test,
			Direct:  true,
		})
	}

	// Test dependencies (indirect)
	for name, version := range elm.TestDependencies.Indirect {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Test,
			Direct:  false,
		})
	}

	return deps, nil
}

// elmPackageJSONParser parses elm-package.json files (Elm 0.18 and earlier).
type elmPackageJSONParser struct{}

type elmPackageJSON struct {
	Dependencies map[string]string `json:"dependencies"`
}

func (p *elmPackageJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var elm elmPackageJSON
	if err := json.Unmarshal(content, &elm); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, version := range elm.Dependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}
