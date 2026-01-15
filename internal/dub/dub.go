package dub

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"
)

func init() {
	core.Register("dub", core.Manifest, &dubJSONParser{}, core.ExactMatch("dub.json"))
	core.Register("dub", core.Manifest, &dubSDLParser{}, core.ExactMatch("dub.sdl"))
}

// dubJSONParser parses dub.json files.
type dubJSONParser struct{}

type dubJSON struct {
	Dependencies map[string]any `json:"dependencies"`
}

func (p *dubJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var dub dubJSON
	if err := json.Unmarshal(content, &dub); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, spec := range dub.Dependencies {
		version := ""
		scope := core.Runtime

		switch s := spec.(type) {
		case string:
			version = s
		case map[string]any:
			if v, ok := s["version"].(string); ok {
				version = v
			}
			if optional, ok := s["optional"].(bool); ok && optional {
				scope = core.Optional
			}
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   scope,
			Direct:  true,
		})
	}

	return deps, nil
}

// dubSDLParser parses dub.sdl files.
type dubSDLParser struct{}

var (
	// dependency "name" version="~>1.0"
	dubSDLDepRegex = regexp.MustCompile(`dependency\s+"([^"]+)"\s+version="([^"]+)"`)
)

func (p *dubSDLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if match := dubSDLDepRegex.FindStringSubmatch(line); match != nil {
			deps = append(deps, core.Dependency{
				Name:    match[1],
				Version: match[2],
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	return deps, nil
}
