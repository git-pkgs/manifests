package cargo

import (
	"github.com/git-pkgs/manifests/internal/core"
	"github.com/BurntSushi/toml"
)

func init() {
	// Cargo.toml - manifest
	core.Register("cargo", core.Manifest, &cargoTomlParser{}, core.ExactMatch("Cargo.toml"))

	// Cargo.lock - lockfile
	core.Register("cargo", core.Lockfile, &cargoLockParser{}, core.ExactMatch("Cargo.lock"))
}

// cargoTomlParser parses Cargo.toml files.
type cargoTomlParser struct{}

func (p *cargoTomlParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var cargo struct {
		Package struct {
			Name string `toml:"name"`
		} `toml:"package"`
		Dependencies      map[string]any `toml:"dependencies"`
		DevDependencies   map[string]any `toml:"dev-dependencies"`
		BuildDependencies map[string]any `toml:"build-dependencies"`
	}

	if _, err := toml.Decode(string(content), &cargo); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	pkgName := cargo.Package.Name

	for name, value := range cargo.Dependencies {
		version := extractCargoVersion(value)
		// Skip local path dependencies
		if isLocalCargoDep(value) {
			continue
		}
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, value := range cargo.DevDependencies {
		version := extractCargoVersion(value)
		if isLocalCargoDep(value) {
			continue
		}
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	for name, value := range cargo.BuildDependencies {
		version := extractCargoVersion(value)
		if isLocalCargoDep(value) {
			continue
		}
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Build,
			Direct:  true,
		})
	}

	// Filter out self-reference
	filtered := deps[:0]
	for _, d := range deps {
		if d.Name != pkgName {
			filtered = append(filtered, d)
		}
	}

	return filtered, nil
}

func extractCargoVersion(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		if ver, ok := v["version"].(string); ok {
			return ver
		}
	}
	return "*"
}

func isLocalCargoDep(value any) bool {
	if m, ok := value.(map[string]any); ok {
		_, hasPath := m["path"]
		return hasPath
	}
	return false
}

// cargoLockParser parses Cargo.lock files using string ops for speed.
type cargoLockParser struct{}

func (p *cargoLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	var currentName, currentVersion, currentSource, currentChecksum string
	inPackage := false

	core.ForEachLine(text, func(line string) bool {
		// Start of a package block
		if line == "[[package]]" {
			// Save previous package if it had a source (not local)
			if inPackage && currentName != "" && currentSource != "" {
				integrity := ""
				if currentChecksum != "" {
					integrity = "sha256-" + currentChecksum
				}
				deps = append(deps, core.Dependency{
					Name:      currentName,
					Version:   currentVersion,
					Scope:   core.Runtime,
					Integrity: integrity,
					Direct:    false,
				})
			}
			currentName = ""
			currentVersion = ""
			currentSource = ""
			currentChecksum = ""
			inPackage = true
			return true
		}

		if !inPackage {
			return true
		}

		if v, ok := core.ExtractQuotedValue(line, "name = "); ok {
			currentName = v
		} else if v, ok := core.ExtractQuotedValue(line, "version = "); ok {
			currentVersion = v
		} else if v, ok := core.ExtractQuotedValue(line, "source = "); ok {
			currentSource = v
		} else if v, ok := core.ExtractQuotedValue(line, "checksum = "); ok {
			currentChecksum = v
		}
		return true
	})

	// Don't forget the last package
	if inPackage && currentName != "" && currentSource != "" {
		integrity := ""
		if currentChecksum != "" {
			integrity = "sha256-" + currentChecksum
		}
		deps = append(deps, core.Dependency{
			Name:      currentName,
			Version:   currentVersion,
			Scope:   core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return deps, nil
}
