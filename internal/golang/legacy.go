package golang

import (
	"encoding/json"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	// Godeps.json - lockfile (godep tool)
	core.Register("golang", core.Lockfile, &godepsJSONParser{}, core.ExactMatch("Godeps.json"))

	// glide.yaml - manifest
	core.Register("golang", core.Manifest, &glideYAMLParser{}, core.ExactMatch("glide.yaml"))

	// glide.lock - lockfile
	core.Register("golang", core.Lockfile, &glideLockParser{}, core.ExactMatch("glide.lock"))

	// Gopkg.toml - manifest (dep tool)
	core.Register("golang", core.Manifest, &gopkgTOMLParser{}, core.ExactMatch("Gopkg.toml"))

	// Gopkg.lock - lockfile (dep tool)
	core.Register("golang", core.Lockfile, &gopkgLockParser{}, core.ExactMatch("Gopkg.lock"))

	// vendor.json - lockfile (govendor tool)
	core.Register("golang", core.Lockfile, &vendorJSONParser{}, core.ExactMatch("vendor.json"))
}

// godepsJSONParser parses Godeps.json files.
type godepsJSONParser struct{}

type godepsJSON struct {
	Deps []struct {
		ImportPath string `json:"ImportPath"`
		Rev        string `json:"Rev"`
		Comment    string `json:"Comment"`
	} `json:"Deps"`
}

func (p *godepsJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var godeps godepsJSON
	if err := json.Unmarshal(content, &godeps); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, dep := range godeps.Deps {
		if dep.ImportPath == "" {
			continue
		}

		// Use Comment (version tag) if available, otherwise Rev
		version := dep.Comment
		if version == "" {
			version = dep.Rev
		}

		deps = append(deps, core.Dependency{
			Name:    dep.ImportPath,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// glideYAMLParser parses glide.yaml files.
type glideYAMLParser struct{}

type glideYAML struct {
	Import []struct {
		Package string `yaml:"package"`
		Version string `yaml:"version"`
	} `yaml:"import"`
}

func (p *glideYAMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var glide glideYAML
	if err := yaml.Unmarshal(content, &glide); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, imp := range glide.Import {
		if imp.Package == "" {
			continue
		}

		deps = append(deps, core.Dependency{
			Name:    imp.Package,
			Version: imp.Version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}

// glideLockParser parses glide.lock files.
type glideLockParser struct{}

type glideLock struct {
	Imports []struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
	} `yaml:"imports"`
}

func (p *glideLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock glideLock
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, imp := range lock.Imports {
		if imp.Name == "" {
			continue
		}

		deps = append(deps, core.Dependency{
			Name:    imp.Name,
			Version: imp.Version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// gopkgTOMLParser parses Gopkg.toml files.
type gopkgTOMLParser struct{}

type gopkgTOML struct {
	Constraint []struct {
		Name    string `toml:"name"`
		Version string `toml:"version"`
		Branch  string `toml:"branch"`
	} `toml:"constraint"`
}

func (p *gopkgTOMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var gopkg gopkgTOML
	if _, err := toml.Decode(string(content), &gopkg); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, c := range gopkg.Constraint {
		if c.Name == "" {
			continue
		}

		// Use version if available, otherwise branch
		version := c.Version
		if version == "" {
			version = c.Branch
		}

		deps = append(deps, core.Dependency{
			Name:    c.Name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}

// gopkgLockParser parses Gopkg.lock files.
type gopkgLockParser struct{}

type gopkgLock struct {
	Projects []struct {
		Name     string `toml:"name"`
		Version  string `toml:"version"`
		Revision string `toml:"revision"`
		Branch   string `toml:"branch"`
	} `toml:"projects"`
}

func (p *gopkgLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock gopkgLock
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, proj := range lock.Projects {
		if proj.Name == "" {
			continue
		}

		// Prefer version, then revision
		version := proj.Version
		if version == "" {
			version = proj.Revision
		}

		deps = append(deps, core.Dependency{
			Name:    proj.Name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// vendorJSONParser parses vendor.json files (govendor).
type vendorJSONParser struct{}

type vendorJSON struct {
	Package []struct {
		Path         string `json:"path"`
		Revision     string `json:"revision"`
		RevisionTime string `json:"revisionTime"`
		ChecksumSHA1 string `json:"checksumSHA1"`
	} `json:"package"`
}

func (p *vendorJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var vendor vendorJSON
	if err := json.Unmarshal(content, &vendor); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, pkg := range vendor.Package {
		if pkg.Path == "" {
			continue
		}

		// Extract base package name (remove subpackage paths)
		name := extractBasePackage(pkg.Path)
		if seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: pkg.Revision,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// extractBasePackage extracts the base package from an import path.
// e.g., "github.com/pkg/errors/stack" -> "github.com/pkg/errors"
func extractBasePackage(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 3 {
		return path
	}
	// For github.com/owner/repo/subpkg, return github.com/owner/repo
	return strings.Join(parts[:3], "/")
}
