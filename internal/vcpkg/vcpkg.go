package vcpkg

import (
	"encoding/json"
	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("vcpkg", core.Manifest, &vcpkgJSONParser{}, core.ExactMatch("vcpkg.json"))
}

// vcpkgJSONParser parses vcpkg.json files.
type vcpkgJSONParser struct{}

type vcpkgJSON struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	VersionString string `json:"version-string"`
	Dependencies  []any  `json:"dependencies"`
}

func (p *vcpkgJSONParser) Parse(filename string, content []byte) (*core.Result, error) {
	var pkg vcpkgJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, dep := range pkg.Dependencies {
		var name string

		switch d := dep.(type) {
		case string:
			name = d
		case map[string]any:
			if n, ok := d["name"].(string); ok {
				name = n
			}
		}

		if name == "" || seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:   name,
			Scope:  core.Runtime,
			Direct: true,
		})
	}

	version := pkg.Version
	if version == "" {
		version = pkg.VersionString
	}
	return &core.Result{Name: pkg.Name, Version: version, Dependencies: deps}, nil
}
