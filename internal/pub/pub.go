package pub

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("pub", core.Manifest, &pubspecYAMLParser{}, core.ExactMatch("pubspec.yaml"))
	core.Register("pub", core.Lockfile, &pubspecLockParser{}, core.ExactMatch("pubspec.lock"))
}

// extractPubspecName extracts package name from "  name:" line
func extractPubspecName(line string) (string, bool) {
	// Must start with exactly 2 spaces and end with colon
	if len(line) < 4 || line[0] != ' ' || line[1] != ' ' || line[2] == ' ' {
		return "", false
	}
	if line[len(line)-1] != ':' {
		return "", false
	}
	return line[2 : len(line)-1], true
}

// extractPubspecVersion extracts version from '    version: "X.Y.Z"' line
func extractPubspecVersion(line string) (string, bool) {
	const prefix = "    version: \""
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	rest := line[len(prefix):]
	end := strings.IndexByte(rest, '"')
	if end < 0 {
		return "", false
	}
	return rest[:end], true
}

// pubspecYAMLParser parses pubspec.yaml files.
type pubspecYAMLParser struct{}

type pubspecYAML struct {
	Dependencies    map[string]any `yaml:"dependencies"`
	DevDependencies map[string]any `yaml:"dev_dependencies"`
}

func (p *pubspecYAMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
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

	return deps, nil
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

// pubspecLockParser parses pubspec.lock files using regex for speed.
type pubspecLockParser struct{}

func (p *pubspecLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	var currentName string
	core.ForEachLine(text, func(line string) bool {
		// Check for package name (2-space indent, ends with colon)
		if name, ok := extractPubspecName(line); ok {
			currentName = name
			return true
		}

		// Check for version (4-space indent)
		if currentName != "" {
			if version, ok := extractPubspecVersion(line); ok {
				deps = append(deps, core.Dependency{
					Name:    currentName,
					Version: version,
					Scope:   core.Runtime,
					Direct:  false,
				})
				currentName = ""
			}
		}
		return true
	})

	return deps, nil
}
