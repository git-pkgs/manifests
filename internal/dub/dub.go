package dub

import (
	"encoding/json"
	"github.com/git-pkgs/manifests/internal/core"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

func init() {
	core.Register("dub", core.Manifest, &dubJSONParser{}, core.ExactMatch("dub.json"))
	core.Register("dub", core.Manifest, &dubSDLParser{}, core.ExactMatch("dub.sdl"))
}

// dubJSONParser parses dub.json files.
type dubJSONParser struct{}

type dubJSON struct {
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Dependencies map[string]any `json:"dependencies"`
}

func (p *dubJSONParser) Parse(filename string, content []byte) (*core.Result, error) {
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

	var recipe map[string]any
	if err := json.Unmarshal(content, &recipe); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
	var scripts map[string][]string
	collectDubScripts(&scripts, "", recipe)
	return &core.Result{Name: dub.Name, Version: dub.Version, Dependencies: deps, Scripts: scripts}, nil
}

func collectDubScripts(scripts *map[string][]string, prefix string, recipe map[string]any) {
	for name, value := range recipe {
		hook, _, _ := strings.Cut(name, "-")
		if isDubHook(hook) {
			for _, commands := range core.StringScripts(map[string]any{name: value}) {
				core.AddScript(scripts, prefix+name, commands...)
			}
		}
	}
	for _, group := range []string{"configurations", "subPackages"} {
		entries, _ := recipe[group].([]any)
		for i, entry := range entries {
			if properties, ok := entry.(map[string]any); ok {
				name, _ := properties["name"].(string)
				if name == "" {
					name = strconv.Itoa(i)
				}
				collectDubScripts(scripts, prefix+group+"/"+url.PathEscape(name)+"/", properties)
			}
		}
	}
	buildTypes, _ := recipe["buildTypes"].(map[string]any)
	for name, entry := range buildTypes {
		if properties, ok := entry.(map[string]any); ok {
			collectDubScripts(scripts, prefix+"buildTypes/"+url.PathEscape(name)+"/", properties)
		}
	}
}

func isDubHook(name string) bool {
	switch name {
	case "preGenerateCommands", "postGenerateCommands", "preBuildCommands", "postBuildCommands", "preRunCommands", "postRunCommands":
		return true
	}
	return false
}

// dubSDLParser parses dub.sdl files.
type dubSDLParser struct{}

var (
	// dependency "name" version="~>1.0"
	dubSDLDepRegex = regexp.MustCompile(`dependency\s+"([^"]+)"\s+version="([^"]+)"`)
)

func (p *dubSDLParser) Parse(filename string, content []byte) (*core.Result, error) {
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

	return &core.Result{Dependencies: deps}, nil
}
