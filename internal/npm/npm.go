package npm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"strings"
)

func init() {
	// package.json - manifest
	core.Register("npm", core.Manifest, &npmPackageJSONParser{}, core.ExactMatch("package.json"))

	// package-lock.json - lockfile
	core.Register("npm", core.Lockfile, &npmPackageLockParser{}, core.ExactMatch("package-lock.json", "npm-shrinkwrap.json"))
}

// npmPackageJSONParser parses package.json files.
type npmPackageJSONParser struct{}

type packageJSON struct {
	Dependencies         map[string]any `json:"dependencies"`
	DevDependencies      map[string]any `json:"devDependencies"`
	OptionalDependencies map[string]any `json:"optionalDependencies"`
	PeerDependencies     map[string]any `json:"peerDependencies"`
}

func (p *npmPackageJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, value := range pkg.Dependencies {
		if isNpmComment(name) {
			continue
		}
		version, ok := value.(string)
		if !ok {
			continue
		}
		realName, realVersion := parseNpmAlias(name, version)
		deps = append(deps, core.Dependency{
			Name:    realName,
			Version: realVersion,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, value := range pkg.DevDependencies {
		if isNpmComment(name) {
			continue
		}
		version, ok := value.(string)
		if !ok {
			continue
		}
		realName, realVersion := parseNpmAlias(name, version)
		deps = append(deps, core.Dependency{
			Name:    realName,
			Version: realVersion,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	for name, value := range pkg.OptionalDependencies {
		if isNpmComment(name) {
			continue
		}
		version, ok := value.(string)
		if !ok {
			continue
		}
		realName, realVersion := parseNpmAlias(name, version)
		deps = append(deps, core.Dependency{
			Name:    realName,
			Version: realVersion,
			Scope:   core.Optional,
			Direct:  true,
		})
	}

	for name, value := range pkg.PeerDependencies {
		if isNpmComment(name) {
			continue
		}
		version, ok := value.(string)
		if !ok {
			continue
		}
		realName, realVersion := parseNpmAlias(name, version)
		deps = append(deps, core.Dependency{
			Name:    realName,
			Version: realVersion,
			Scope:   core.Runtime, // peer dependencies are runtime requirements
			Direct:  true,
		})
	}

	return deps, nil
}

// isNpmComment checks if a dependency name is actually a comment.
// npm allows keys starting with "//" as comments in package.json.
func isNpmComment(name string) bool {
	return strings.HasPrefix(name, "//")
}

// parseNpmAlias handles npm alias syntax: "alias-name": "npm:@scope/real-name@version"
func parseNpmAlias(name, version string) (string, string) {
	if strings.HasPrefix(version, "npm:") {
		// Format: npm:@scope/package@version or npm:package@version
		aliased := strings.TrimPrefix(version, "npm:")
		if idx := strings.LastIndex(aliased, "@"); idx > 0 {
			return aliased[:idx], aliased[idx+1:]
		}
		return aliased, ""
	}
	return name, version
}

// npmPackageLockParser parses package-lock.json files.
type npmPackageLockParser struct{}

// packageLockJSON supports both v1/v2 and v3 lockfile formats.
type packageLockJSON struct {
	LockfileVersion int `json:"lockfileVersion"`

	// v1/v2 format
	Dependencies map[string]packageLockDep `json:"dependencies"`

	// v3 format (also in v2 for backwards compat)
	Packages map[string]packageLockPackage `json:"packages"`
}

type packageLockDep struct {
	Version      string                       `json:"version"`
	Resolved     string                       `json:"resolved"`
	Integrity    string                       `json:"integrity"`
	Dev          bool                         `json:"dev"`
	Optional     bool                         `json:"optional"`
	Dependencies map[string]packageLockDep    `json:"dependencies"`
}

type packageLockPackage struct {
	Version      string `json:"version"`
	Resolved     string `json:"resolved"`
	Integrity    string `json:"integrity"`
	Dev          bool   `json:"dev"`
	Optional     bool   `json:"optional"`
	DevOptional  bool   `json:"devOptional"`
}

func (p *npmPackageLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock packageLockJSON
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	// v3 format uses "packages" with paths as keys
	if lock.LockfileVersion >= 2 && len(lock.Packages) > 0 {
		return parsePackageLockV3(lock.Packages), nil
	}

	// v1 format uses "dependencies"
	return parsePackageLockV1(lock.Dependencies), nil
}

func parsePackageLockV1(deps map[string]packageLockDep) []core.Dependency {
	var result []core.Dependency
	for name, dep := range deps {
		scope := core.Runtime
		if dep.Dev {
			scope = core.Development
		} else if dep.Optional {
			scope = core.Optional
		}

		result = append(result, core.Dependency{
			Name:      name,
			Version:   dep.Version,
			Scope:     scope,
			Integrity: dep.Integrity,
			Direct:    false, // lockfiles don't distinguish direct vs transitive in v1
		})

		// Recursively add nested dependencies
		if len(dep.Dependencies) > 0 {
			nested := parsePackageLockV1(dep.Dependencies)
			result = append(result, nested...)
		}
	}
	return result
}

func parsePackageLockV3(packages map[string]packageLockPackage) []core.Dependency {
	var result []core.Dependency
	for path, pkg := range packages {
		// Skip the root package (empty path or "")
		if path == "" {
			continue
		}

		// Extract package name from path
		// Path format: node_modules/pkg or node_modules/@scope/pkg
		// or nested: node_modules/pkg/node_modules/nested
		name := extractPackageName(path)
		if name == "" {
			continue
		}

		scope := core.Runtime
		if pkg.Dev || pkg.DevOptional {
			scope = core.Development
		} else if pkg.Optional {
			scope = core.Optional
		}

		// Direct dependencies are in node_modules/ directly (not nested)
		direct := !strings.Contains(strings.TrimPrefix(path, "node_modules/"), "node_modules/")

		result = append(result, core.Dependency{
			Name:      name,
			Version:   pkg.Version,
			Scope:     scope,
			Integrity: pkg.Integrity,
			Direct:    direct,
		})
	}
	return result
}

// extractPackageName extracts the package name from a node_modules path.
func extractPackageName(path string) string {
	// Remove leading node_modules/
	path = strings.TrimPrefix(path, "node_modules/")

	// For nested deps, get the last package in the chain
	if idx := strings.LastIndex(path, "node_modules/"); idx >= 0 {
		path = path[idx+len("node_modules/"):]
	}

	// Handle scoped packages (@scope/name)
	if strings.HasPrefix(path, "@") {
		// @scope/name - return the full scoped name
		parts := strings.SplitN(path, "/", 3)
		if len(parts) >= 2 {
			return parts[0] + "/" + parts[1]
		}
	}

	// Regular package - just return the first path component
	if idx := strings.Index(path, "/"); idx >= 0 {
		return path[:idx]
	}
	return path
}
