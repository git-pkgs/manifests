package npm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
)

func init() {
	core.Register("bower", core.Manifest, &bowerParser{}, core.ExactMatch("bower.json"))
}

// bowerParser parses bower.json files.
type bowerParser struct{}

type bowerJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (p *bowerParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var bower bowerJSON
	if err := json.Unmarshal(content, &bower); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, version := range bower.Dependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, version := range bower.DevDependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return deps, nil
}
