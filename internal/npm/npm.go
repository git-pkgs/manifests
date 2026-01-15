package npm

import (
	"encoding/json"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	// package.json - manifest
	core.Register("npm", core.Manifest, &npmPackageJSONParser{}, core.ExactMatch("package.json"))

	// package-lock.json - lockfile
	core.Register("npm", core.Lockfile, &npmPackageLockParser{}, core.ExactMatch("package-lock.json", "npm-shrinkwrap.json"))

	// npm-ls.json - lockfile (output from npm ls --json)
	core.Register("npm", core.Lockfile, &npmLsParser{}, core.ExactMatch("npm-ls.json"))
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
}

type packageLockDep struct {
	Version      string                    `json:"version"`
	Resolved     string                    `json:"resolved"`
	Integrity    string                    `json:"integrity"`
	Dev          bool                      `json:"dev"`
	Optional     bool                      `json:"optional"`
	Dependencies map[string]packageLockDep `json:"dependencies"`
}

func (p *npmPackageLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	// Quick check for lockfile version to determine parsing strategy
	// v3 (lockfileVersion >= 2 with packages) uses line-based parsing
	// v1 uses JSON parsing for nested dependencies
	header := string(content[:min(200, len(content))])

	// v2+ with packages section uses line-based v3 parsing
	if strings.Contains(header, `"lockfileVersion": 3`) ||
		(strings.Contains(header, `"lockfileVersion": 2`) && strings.Contains(string(content[:min(600, len(content))]), `"packages"`)) {
		return parsePackageLockV3Lines(content), nil
	}

	// v1 format uses JSON (nested dependencies make line parsing complex)
	var lock packageLockJSON
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
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
			Direct:    false,
		})

		// Recursively add nested dependencies
		if len(dep.Dependencies) > 0 {
			nested := parsePackageLockV1(dep.Dependencies)
			result = append(result, nested...)
		}
	}
	return result
}

// parsePackageLockV3Lines parses v3 format using line-based parsing.
// Format: "packages": { "node_modules/name": { "version": "x", ... } }
func parsePackageLockV3Lines(content []byte) []core.Dependency {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	inPackages := false
	var currentPath string
	var currentVersion string
	var currentIntegrity string
	var currentDev bool
	var currentOptional bool
	var currentDevOptional bool
	var currentLink bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect "packages": section start
		if strings.HasPrefix(trimmed, `"packages"`) {
			inPackages = true
			continue
		}

		if !inPackages {
			continue
		}

		// Detect end of packages section (closing brace followed by new top-level key)
		// packages section ends with `  },` at indent 2
		if (line == "  }," || line == "  }") && strings.HasPrefix(trimmed, "}") {
			break
		}

		// Package path line: "node_modules/name": {
		if strings.HasSuffix(trimmed, ": {") || strings.HasSuffix(trimmed, ":{") {
			// Save previous package (include links even without version)
			if currentPath != "" && (currentVersion != "" || currentLink) {
				name := extractPackageName(currentPath)
				if name != "" {
					scope := core.Runtime
					if currentDev || currentDevOptional {
						scope = core.Development
					} else if currentOptional {
						scope = core.Optional
					}
					direct := !strings.Contains(strings.TrimPrefix(currentPath, "node_modules/"), "node_modules/")
					deps = append(deps, core.Dependency{
						Name:      name,
						Version:   currentVersion,
						Scope:     scope,
						Integrity: currentIntegrity,
						Direct:    direct,
					})
				}
			}
			// Extract path
			start := strings.IndexByte(trimmed, '"')
			end := strings.IndexByte(trimmed[start+1:], '"')
			if start >= 0 && end > 0 {
				currentPath = trimmed[start+1 : start+1+end]
			}
			currentVersion = ""
			currentIntegrity = ""
			currentDev = false
			currentOptional = false
			currentDevOptional = false
			currentLink = false
			continue
		}

		// Version line
		if strings.HasPrefix(trimmed, `"version"`) {
			if v := extractJSONStringValue(trimmed); v != "" {
				currentVersion = v
			}
			continue
		}

		// Integrity line
		if strings.HasPrefix(trimmed, `"integrity"`) {
			if v := extractJSONStringValue(trimmed); v != "" {
				currentIntegrity = v
			}
			continue
		}

		// Dev/optional/link flags
		if strings.HasPrefix(trimmed, `"dev": true`) {
			currentDev = true
		}
		if strings.HasPrefix(trimmed, `"optional": true`) {
			currentOptional = true
		}
		if strings.HasPrefix(trimmed, `"devOptional": true`) {
			currentDevOptional = true
		}
		if strings.HasPrefix(trimmed, `"link": true`) {
			currentLink = true
		}
	}

	// Don't forget the last package
	if currentPath != "" && (currentVersion != "" || currentLink) {
		name := extractPackageName(currentPath)
		if name != "" {
			scope := core.Runtime
			if currentDev || currentDevOptional {
				scope = core.Development
			} else if currentOptional {
				scope = core.Optional
			}
			direct := !strings.Contains(strings.TrimPrefix(currentPath, "node_modules/"), "node_modules/")
			deps = append(deps, core.Dependency{
				Name:      name,
				Version:   currentVersion,
				Scope:     scope,
				Integrity: currentIntegrity,
				Direct:    direct,
			})
		}
	}

	return deps
}

// extractJSONStringValue extracts the string value from a JSON line like: "key": "value"
func extractJSONStringValue(line string) string {
	// Find the colon
	colonIdx := strings.IndexByte(line, ':')
	if colonIdx < 0 {
		return ""
	}
	rest := line[colonIdx+1:]
	// Find the opening quote
	start := strings.IndexByte(rest, '"')
	if start < 0 {
		return ""
	}
	// Find the closing quote
	end := strings.IndexByte(rest[start+1:], '"')
	if end < 0 {
		return ""
	}
	return rest[start+1 : start+1+end]
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

// npmLsParser parses npm-ls.json files (output from npm ls --json).
type npmLsParser struct{}

type npmLsJSON struct {
	Dependencies map[string]npmLsDep `json:"dependencies"`
}

type npmLsDep struct {
	Version      string                 `json:"version"`
	Resolved     string                 `json:"resolved"`
	Integrity    string                 `json:"integrity"`
	Dev          bool                   `json:"dev"`
	Dependencies map[string]npmLsDep    `json:"dependencies"`
}

func (p *npmLsParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var ls npmLsJSON
	if err := json.Unmarshal(content, &ls); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	return parseNpmLsDeps(ls.Dependencies, make(map[string]bool)), nil
}

func parseNpmLsDeps(deps map[string]npmLsDep, seen map[string]bool) []core.Dependency {
	var result []core.Dependency

	for name, dep := range deps {
		if seen[name] {
			continue
		}
		seen[name] = true

		scope := core.Runtime
		if dep.Dev {
			scope = core.Development
		}

		result = append(result, core.Dependency{
			Name:      name,
			Version:   dep.Version,
			Scope:     scope,
			Integrity: dep.Integrity,
			Direct:    false,
		})

		// Recursively add nested dependencies
		if len(dep.Dependencies) > 0 {
			nested := parseNpmLsDeps(dep.Dependencies, seen)
			result = append(result, nested...)
		}
	}

	return result
}
