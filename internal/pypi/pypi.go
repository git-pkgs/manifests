package pypi

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

func init() {
	// requirements.txt variants - manifests
	core.Register("pypi", core.Manifest, &requirementsTxtParser{},
		core.AnyMatch(
			core.ExactMatch("requirements.txt"),
			core.SuffixMatch("-requirements.txt", "_requirements.txt", ".requirements.txt"),
			core.PrefixMatch("requirements-", "requirements_", "requirements."),
		))

	// Pipfile - manifest
	core.Register("pypi", core.Manifest, &pipfileParser{}, core.ExactMatch("Pipfile"))

	// Pipfile.lock - lockfile
	core.Register("pypi", core.Lockfile, &pipfileLockParser{}, core.ExactMatch("Pipfile.lock"))

	// pyproject.toml - manifest (Poetry/PEP 621)
	core.Register("pypi", core.Manifest, &pyprojectParser{}, core.ExactMatch("pyproject.toml"))

	// poetry.lock - lockfile
	core.Register("pypi", core.Lockfile, &poetryLockParser{}, core.ExactMatch("poetry.lock"))

	// pdm.lock - lockfile
	core.Register("pypi", core.Lockfile, &pdmLockParser{}, core.ExactMatch("pdm.lock"))

	// uv.lock - lockfile
	core.Register("pypi", core.Lockfile, &uvLockParser{}, core.ExactMatch("uv.lock"))
}

// requirementsTxtParser parses requirements.txt files.
type requirementsTxtParser struct{}

var (
	// pkg==1.0.0 or pkg>=1.0.0 or pkg~=1.0.0
	requirementRegex = regexp.MustCompile(`^([a-zA-Z0-9_.-]+(?:\[[^\]]+\])?)\s*(==|>=|<=|~=|!=|>|<)?(.*)`)
)

func (p *requirementsTxtParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Remove comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)

		// Skip empty lines and options
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}

		if match := requirementRegex.FindStringSubmatch(line); match != nil {
			name := match[1]
			// Remove extras bracket if present
			if idx := strings.Index(name, "["); idx >= 0 {
				name = name[:idx]
			}

			version := ""
			if match[2] != "" && match[3] != "" {
				version = match[2] + match[3]
			}

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// pipfileParser parses Pipfile (TOML format).
type pipfileParser struct{}

func (p *pipfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var pipfile struct {
		Packages    map[string]any `toml:"packages"`
		DevPackages map[string]any `toml:"dev-packages"`
	}

	if _, err := toml.Decode(string(content), &pipfile); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, value := range pipfile.Packages {
		version := extractPipfileVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, value := range pipfile.DevPackages {
		version := extractPipfileVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return deps, nil
}

func extractPipfileVersion(value any) string {
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

// pipfileLockParser parses Pipfile.lock (JSON format).
type pipfileLockParser struct{}

type pipfileLock struct {
	Default map[string]pipfileLockDep `json:"default"`
	Develop map[string]pipfileLockDep `json:"develop"`
}

type pipfileLockDep struct {
	Version string   `json:"version"`
	Hashes  []string `json:"hashes"`
}

func (p *pipfileLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock pipfileLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, dep := range lock.Default {
		version := strings.TrimPrefix(dep.Version, "==")
		integrity := ""
		if len(dep.Hashes) > 0 {
			// Use first hash, convert to SRI format
			integrity = convertPythonHash(dep.Hashes[0])
		}
		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:   core.Runtime,
			Integrity: integrity,
			Direct:    false, // Pipfile.lock doesn't distinguish
		})
	}

	for name, dep := range lock.Develop {
		version := strings.TrimPrefix(dep.Version, "==")
		integrity := ""
		if len(dep.Hashes) > 0 {
			integrity = convertPythonHash(dep.Hashes[0])
		}
		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:   core.Development,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return deps, nil
}

// convertPythonHash converts a Python hash (sha256:...) to SRI format (sha256-...).
func convertPythonHash(h string) string {
	if strings.HasPrefix(h, "sha256:") {
		return "sha256-" + strings.TrimPrefix(h, "sha256:")
	}
	if strings.HasPrefix(h, "sha512:") {
		return "sha512-" + strings.TrimPrefix(h, "sha512:")
	}
	return h
}

// pyprojectParser parses pyproject.toml (Poetry format).
type pyprojectParser struct{}

func (p *pyprojectParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var pyproject struct {
		Tool struct {
			Poetry struct {
				Dependencies    map[string]any `toml:"dependencies"`
				DevDependencies map[string]any `toml:"dev-dependencies"`
				Group           map[string]struct {
					Dependencies map[string]any `toml:"dependencies"`
				} `toml:"group"`
			} `toml:"poetry"`
		} `toml:"tool"`
		Project struct {
			Dependencies         []string `toml:"dependencies"`
			OptionalDependencies map[string][]string `toml:"optional-dependencies"`
		} `toml:"project"`
	}

	if _, err := toml.Decode(string(content), &pyproject); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	// Poetry format
	for name, value := range pyproject.Tool.Poetry.Dependencies {
		if name == "python" {
			continue
		}
		version := extractPoetryVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, value := range pyproject.Tool.Poetry.DevDependencies {
		version := extractPoetryVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	// Poetry group dependencies
	for groupName, group := range pyproject.Tool.Poetry.Group {
		scope := core.Runtime
		if groupName == "dev" || groupName == "development" {
			scope = core.Development
		} else if groupName == "test" {
			scope = core.Test
		}

		for name, value := range group.Dependencies {
			version := extractPoetryVersion(value)
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
		}
	}

	// PEP 621 format
	for _, dep := range pyproject.Project.Dependencies {
		name, version := parsePEP508(dep)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}

func extractPoetryVersion(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		if ver, ok := v["version"].(string); ok {
			return ver
		}
	case []any:
		// Multiple version constraints
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				if ver, ok := m["version"].(string); ok {
					return ver
				}
			}
		}
	}
	return "*"
}

func parsePEP508(dep string) (string, string) {
	// Simple parsing for "pkg>=1.0.0" or "pkg[extra]>=1.0.0"
	dep = strings.TrimSpace(dep)

	// Find where version spec starts
	for i, c := range dep {
		if c == '>' || c == '<' || c == '=' || c == '~' || c == '!' || c == ';' {
			name := strings.TrimSpace(dep[:i])
			// Remove extras
			if idx := strings.Index(name, "["); idx >= 0 {
				name = name[:idx]
			}
			version := ""
			if c != ';' {
				// Find end of version (before ;)
				rest := dep[i:]
				if idx := strings.Index(rest, ";"); idx >= 0 {
					rest = rest[:idx]
				}
				version = strings.TrimSpace(rest)
			}
			return name, version
		}
	}

	// No version spec
	name := dep
	if idx := strings.Index(name, "["); idx >= 0 {
		name = name[:idx]
	}
	return name, ""
}

// poetryLockParser parses poetry.lock files.
type poetryLockParser struct{}

type poetryLockFile struct {
	Package []poetryLockPackage `toml:"package"`
}

type poetryLockPackage struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Groups  []string `toml:"groups"`
	Files   []struct {
		File string `toml:"file"`
		Hash string `toml:"hash"`
	} `toml:"files"`
}

func (p *poetryLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock poetryLockFile
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Package {
		scope := core.Runtime

		// Determine scope from groups
		for _, g := range pkg.Groups {
			if g == "dev" || g == "development" {
				scope = core.Development
				break
			}
			if g == "test" {
				scope = core.Test
				break
			}
		}

		integrity := ""
		if len(pkg.Files) > 0 {
			integrity = convertPythonHash(pkg.Files[0].Hash)
		}

		deps = append(deps, core.Dependency{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Scope:     scope,
			Integrity: integrity,
			Direct:    false, // poetry.lock doesn't distinguish direct
		})
	}

	return deps, nil
}

// pdmLockParser parses pdm.lock files.
type pdmLockParser struct{}

type pdmLockFile struct {
	Package []pdmLockPackage `toml:"package"`
}

type pdmLockPackage struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Groups  []string `toml:"groups"`
	Files   []struct {
		File string `toml:"file"`
		Hash string `toml:"hash"`
	} `toml:"files"`
}

func (p *pdmLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock pdmLockFile
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Package {
		scope := core.Runtime

		// Check if dev dependency
		for _, g := range pkg.Groups {
			if g == "dev" || g == "development" {
				scope = core.Development
				break
			}
		}

		integrity := ""
		if len(pkg.Files) > 0 && pkg.Files[0].Hash != "" {
			integrity = convertPythonHash(pkg.Files[0].Hash)
		}

		deps = append(deps, core.Dependency{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Scope:     scope,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return deps, nil
}

// uvLockParser parses uv.lock files.
type uvLockParser struct{}

type uvLockFile struct {
	Package []uvLockPackage `toml:"package"`
}

type uvLockPackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Sdist   struct {
		Hash string `toml:"hash"`
	} `toml:"sdist"`
	Wheels []struct {
		Hash string `toml:"hash"`
	} `toml:"wheels"`
}

func (p *uvLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock uvLockFile
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Package {
		integrity := ""
		// Prefer sdist hash, fall back to first wheel hash
		if pkg.Sdist.Hash != "" {
			integrity = convertPythonHash(pkg.Sdist.Hash)
		} else if len(pkg.Wheels) > 0 && pkg.Wheels[0].Hash != "" {
			integrity = convertPythonHash(pkg.Wheels[0].Hash)
		}

		deps = append(deps, core.Dependency{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Scope:     core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return deps, nil
}
