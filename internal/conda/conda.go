package conda

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("conda", core.Manifest, &condaEnvParser{}, core.ExactMatch("environment.yml"))
	core.Register("conda", core.Manifest, &condaEnvParser{}, core.ExactMatch("environment.yaml"))
	core.Register("conda", core.Lockfile, &condaLockParser{}, core.ExactMatch("conda-lock.yml"))
}

// condaEnvParser parses Conda environment.yml files.
type condaEnvParser struct{}

type condaEnvironment struct {
	Name         string `yaml:"name"`
	Channels     []string `yaml:"channels"`
	Dependencies []any    `yaml:"dependencies"`
}

func (p *condaEnvParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var env condaEnvironment
	if err := yaml.Unmarshal(content, &env); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, dep := range env.Dependencies {
		switch d := dep.(type) {
		case string:
			// Regular conda dependency: "name", "name=version", or "name=version=build"
			name, version := parseCondaSpec(d)
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		case map[string]any:
			// Skip nested pip dependencies - they belong to pypi ecosystem
		}
	}

	return deps, nil
}

// parseCondaSpec parses a Conda dependency spec like "name", "name=version", or "name=version=build".
func parseCondaSpec(spec string) (name, version string) {
	parts := strings.Split(spec, "=")
	name = parts[0]
	if len(parts) > 1 {
		version = parts[1]
	}
	return name, version
}

// condaLockParser parses conda-lock.yml files.
type condaLockParser struct{}

type condaLockFile struct {
	Version  int               `yaml:"version"`
	Package  []condaLockPkg    `yaml:"package"`
}

type condaLockPkg struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Manager  string            `yaml:"manager"`
	Platform string            `yaml:"platform"`
	URL      string            `yaml:"url"`
	Hash     condaLockHash     `yaml:"hash"`
	Category string            `yaml:"category"`
	Optional bool              `yaml:"optional"`
}

type condaLockHash struct {
	MD5    string `yaml:"md5"`
	SHA256 string `yaml:"sha256"`
}

func (p *condaLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock condaLockFile
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, pkg := range lock.Package {
		// Skip pip packages - they belong to pypi ecosystem
		if pkg.Manager == "pip" {
			continue
		}

		// Deduplicate across platforms
		if seen[pkg.Name] {
			continue
		}
		seen[pkg.Name] = true

		scope := core.Runtime
		if pkg.Category == "dev" {
			scope = core.Development
		}

		integrity := ""
		if pkg.Hash.SHA256 != "" {
			integrity = "sha256-" + pkg.Hash.SHA256
		} else if pkg.Hash.MD5 != "" {
			integrity = "md5-" + pkg.Hash.MD5
		}

		// Extract channel URL from package URL
		registryURL := ""
		if strings.Contains(pkg.URL, "conda.anaconda.org") {
			// Extract channel: https://conda.anaconda.org/conda-forge/linux-64/...
			parts := strings.Split(pkg.URL, "/")
			if len(parts) >= 4 {
				registryURL = strings.Join(parts[:4], "/")
			}
		}

		deps = append(deps, core.Dependency{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Scope:       scope,
			Integrity:   integrity,
			Direct:      false,
			RegistryURL: registryURL,
		})
	}

	return deps, nil
}
