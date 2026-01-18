package composer

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
)

func init() {
	core.Register("composer", core.Manifest, &composerJSONParser{}, core.ExactMatch("composer.json"))
	core.Register("composer", core.Lockfile, &composerLockParser{}, core.ExactMatch("composer.lock"))
}

// composerJSONParser parses composer.json files.
type composerJSONParser struct{}

type composerJSON struct {
	Require    map[string]string `json:"require"`
	RequireDev map[string]string `json:"require-dev"`
}

func (p *composerJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var composer composerJSON
	if err := json.Unmarshal(content, &composer); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, version := range composer.Require {
		// Skip PHP version requirement
		if name == "php" || name == "php-64bit" {
			continue
		}
		// Skip ext- requirements
		if len(name) > 4 && name[:4] == "ext-" {
			continue
		}
		// Skip lib- requirements
		if len(name) > 4 && name[:4] == "lib-" {
			continue
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, version := range composer.RequireDev {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return deps, nil
}

// composerLockParser parses composer.lock files.
type composerLockParser struct{}

type composerLock struct {
	Packages    []composerPackage `json:"packages"`
	PackagesDev []composerPackage `json:"packages-dev"`
}

type composerPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Dist    struct {
		URL     string `json:"url"`
		SHA     string `json:"shasum"`
		SHA256  string `json:"sha256"`
	} `json:"dist"`
}

func (p *composerLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock composerLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Packages {
		integrity := ""
		if pkg.Dist.SHA256 != "" {
			integrity = "sha256-" + pkg.Dist.SHA256
		} else if pkg.Dist.SHA != "" {
			integrity = "sha1-" + pkg.Dist.SHA
		}

		deps = append(deps, core.Dependency{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Scope:       core.Runtime,
			Integrity:   integrity,
			Direct:      false, // composer.lock doesn't distinguish
			RegistryURL: pkg.Dist.URL,
		})
	}

	for _, pkg := range lock.PackagesDev {
		integrity := ""
		if pkg.Dist.SHA256 != "" {
			integrity = "sha256-" + pkg.Dist.SHA256
		} else if pkg.Dist.SHA != "" {
			integrity = "sha1-" + pkg.Dist.SHA
		}

		deps = append(deps, core.Dependency{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Scope:       core.Development,
			Integrity:   integrity,
			Direct:      false,
			RegistryURL: pkg.Dist.URL,
		})
	}

	return deps, nil
}
