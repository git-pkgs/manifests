package haxelib

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
)

func init() {
	core.Register("haxelib", core.Manifest, &haxelibJSONParser{}, core.ExactMatch("haxelib.json"))
}

// haxelibJSONParser parses haxelib.json files.
type haxelibJSONParser struct{}

type haxelibJSON struct {
	Dependencies map[string]string `json:"dependencies"`
}

func (p *haxelibJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var haxelib haxelibJSON
	if err := json.Unmarshal(content, &haxelib); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, version := range haxelib.Dependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}
