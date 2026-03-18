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
	const headerPeekSize = 200
	const packagesPeekSize = 600
	header := string(content[:min(headerPeekSize, len(content))])

	// v2+ with packages section uses line-based v3 parsing
	if strings.Contains(header, `"lockfileVersion": 3`) ||
		(strings.Contains(header, `"lockfileVersion": 2`) && strings.Contains(string(content[:min(packagesPeekSize, len(content))]), `"packages"`)) {
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
			Name:        name,
			Version:     dep.Version,
			Scope:       scope,
			Integrity:   dep.Integrity,
			Direct:      false,
			RegistryURL: dep.Resolved,
		})

		// Recursively add nested dependencies
		if len(dep.Dependencies) > 0 {
			nested := parsePackageLockV1(dep.Dependencies)
			result = append(result, nested...)
		}
	}
	return result
}

// v3PackageEntry holds the state accumulated while parsing a single package
// entry in the v3 lockfile format.
type v3PackageEntry struct {
	path        string
	version     string
	integrity   string
	resolved    string
	dev         bool
	optional    bool
	devOptional bool
	link        bool
}

func (e *v3PackageEntry) reset(path string) {
	e.path = path
	e.version = ""
	e.integrity = ""
	e.resolved = ""
	e.dev = false
	e.optional = false
	e.devOptional = false
	e.link = false
}

func (e *v3PackageEntry) hasContent() bool {
	return e.path != "" && (e.version != "" || e.link)
}

func (e *v3PackageEntry) toDependency() (core.Dependency, bool) {
	name := extractPackageName(e.path)
	if name == "" {
		return core.Dependency{}, false
	}
	scope := core.Runtime
	if e.dev || e.devOptional {
		scope = core.Development
	} else if e.optional {
		scope = core.Optional
	}
	direct := !strings.Contains(strings.TrimPrefix(e.path, "node_modules/"), "node_modules/")
	return core.Dependency{
		Name:        name,
		Version:     e.version,
		Scope:       scope,
		Integrity:   e.integrity,
		Direct:      direct,
		RegistryURL: e.resolved,
	}, true
}

// updateFromLine reads a trimmed line and updates the entry's fields.
// Returns true if the line was consumed.
func (e *v3PackageEntry) updateFromLine(trimmed string) bool {
	switch {
	case strings.HasPrefix(trimmed, `"version"`):
		if v := extractJSONStringValue(trimmed); v != "" {
			e.version = v
		}
	case strings.HasPrefix(trimmed, `"integrity"`):
		if v := extractJSONStringValue(trimmed); v != "" {
			e.integrity = v
		}
	case strings.HasPrefix(trimmed, `"resolved"`):
		if v := extractJSONStringValue(trimmed); v != "" {
			e.resolved = v
		}
	case strings.HasPrefix(trimmed, `"dev": true`):
		e.dev = true
	case strings.HasPrefix(trimmed, `"optional": true`):
		e.optional = true
	case strings.HasPrefix(trimmed, `"devOptional": true`):
		e.devOptional = true
	case strings.HasPrefix(trimmed, `"link": true`):
		e.link = true
	default:
		return false
	}
	return true
}

// extractQuotedPath pulls the quoted string from a line like `"node_modules/foo": {`.
func extractQuotedPath(trimmed string) string {
	start := strings.IndexByte(trimmed, '"')
	if start < 0 {
		return ""
	}
	end := strings.IndexByte(trimmed[start+1:], '"')
	if end <= 0 {
		return ""
	}
	return trimmed[start+1 : start+1+end]
}

// isPackagePathLine returns true for lines like `"node_modules/name": {`.
func isPackagePathLine(trimmed string) bool {
	return strings.HasSuffix(trimmed, ": {") || strings.HasSuffix(trimmed, ":{")
}

// isPackagesSectionEnd detects the closing brace of the "packages" object.
func isPackagesSectionEnd(line, trimmed string) bool {
	return (line == "  }," || line == "  }") && strings.HasPrefix(trimmed, "}")
}

// parsePackageLockV3Lines parses v3 format using line-based parsing.
// Format: "packages": { "node_modules/name": { "version": "x", ... } }
func parsePackageLockV3Lines(content []byte) []core.Dependency {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	inPackages := false
	var entry v3PackageEntry

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if !inPackages {
			if strings.HasPrefix(trimmed, `"packages"`) {
				inPackages = true
			}
			continue
		}

		if isPackagesSectionEnd(line, trimmed) {
			break
		}

		if isPackagePathLine(trimmed) {
			if entry.hasContent() {
				if dep, ok := entry.toDependency(); ok {
					deps = append(deps, dep)
				}
			}
			entry.reset(extractQuotedPath(trimmed))
			continue
		}

		entry.updateFromLine(trimmed)
	}

	// Don't forget the last package
	if entry.hasContent() {
		if dep, ok := entry.toDependency(); ok {
			deps = append(deps, dep)
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
	const scopeAndName = 2
	if strings.HasPrefix(path, "@") {
		// @scope/name - return the full scoped name
		const scopedParts = 3 // @scope/name/rest
		parts := strings.SplitN(path, "/", scopedParts)
		if len(parts) >= scopeAndName {
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
			Name:        name,
			Version:     dep.Version,
			Scope:       scope,
			Integrity:   dep.Integrity,
			Direct:      false,
			RegistryURL: dep.Resolved,
		})

		// Recursively add nested dependencies
		if len(dep.Dependencies) > 0 {
			nested := parseNpmLsDeps(dep.Dependencies, seen)
			result = append(result, nested...)
		}
	}

	return result
}
