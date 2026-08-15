package pub

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("pub", core.Manifest, &pubspecYAMLParser{}, core.ExactMatch("pubspec.yaml"))
	core.Register("pub", core.Lockfile, &pubspecLockParser{}, core.ExactMatch("pubspec.lock"))
}

// pubspecYAMLParser parses pubspec.yaml files.
type pubspecYAMLParser struct{}

type pubspecYAML struct {
	Name            string         `yaml:"name"`
	Version         string         `yaml:"version"`
	Dependencies    map[string]any `yaml:"dependencies"`
	DevDependencies map[string]any `yaml:"dev_dependencies"`
}

func (p *pubspecYAMLParser) Parse(filename string, content []byte) (*core.Result, error) {
	var pubspec pubspecYAML
	if err := yaml.Unmarshal(content, &pubspec); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, spec := range pubspec.Dependencies {
		version := parsePubVersion(spec)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, spec := range pubspec.DevDependencies {
		version := parsePubVersion(spec)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return &core.Result{Name: pubspec.Name, Version: pubspec.Version, Dependencies: deps}, nil
}

// parsePubVersion extracts version from a pubspec dependency spec.
func parsePubVersion(spec any) string {
	switch v := spec.(type) {
	case string:
		return v
	case map[string]any:
		if ver, ok := v["version"].(string); ok {
			return ver
		}
	}
	return ""
}

// pubspecLockParser parses pubspec.lock files.
type pubspecLockParser struct{}

type pubspecLock struct {
	Packages map[string]pubspecLockedPackage `yaml:"packages"`
}

type pubspecLockedPackage struct {
	Dependency  string    `yaml:"dependency"`
	Description yaml.Node `yaml:"description"`
	Source      string    `yaml:"source"`
	Version     string    `yaml:"version"`
}

type pubspecLockDescription struct {
	SHA256 string `yaml:"sha256"`
	URL    string `yaml:"url"`
}

func (p *pubspecLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock pubspecLock
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	deps := make([]core.Dependency, 0, len(lock.Packages))
	for name, pkg := range lock.Packages {
		var description pubspecLockDescription
		if pkg.Description.Kind == yaml.MappingNode {
			if err := pkg.Description.Decode(&description); err != nil {
				return nil, &core.ParseError{Filename: filename, Err: err}
			}
		}

		scope := core.Runtime
		if pkg.Dependency == "direct dev" {
			scope = core.Development
		}

		integrity := ""
		registryURL := ""
		if pkg.Source == "hosted" {
			if description.SHA256 != "" {
				integrity = "sha256-" + strings.TrimPrefix(description.SHA256, "sha256-")
			}
			registryURL = description.URL
		}

		deps = append(deps, core.Dependency{
			Name:        name,
			Version:     pkg.Version,
			Scope:       scope,
			Integrity:   integrity,
			Direct:      strings.HasPrefix(pkg.Dependency, "direct "),
			RegistryURL: registryURL,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}
