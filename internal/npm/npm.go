package npm

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"errors"
	"iter"
	"net/url"
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
	Name                 string         `json:"name"`
	Version              string         `json:"version"`
	License              any            `json:"license"`
	Licenses             []npmLicense   `json:"licenses"`
	Scripts              map[string]any `json:"scripts"`
	Dependencies         map[string]any `json:"dependencies"`
	DevDependencies      map[string]any `json:"devDependencies"`
	OptionalDependencies map[string]any `json:"optionalDependencies"`
	PeerDependencies     map[string]any `json:"peerDependencies"`
}

type npmLicense struct {
	Type string `json:"type"`
}

func (p *npmPackageJSONParser) Parse(filename string, content []byte) (*core.Result, error) {
	var pkg packageJSON
	if err := json.Unmarshal(content, &pkg); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	collectNpmDeclarations(&deps, &declarations, "dependencies", pkg.Dependencies, core.Runtime)
	collectNpmDeclarations(&deps, &declarations, "devDependencies", pkg.DevDependencies, core.Development)
	collectNpmDeclarations(&deps, &declarations, "optionalDependencies", pkg.OptionalDependencies, core.Optional)
	collectNpmDeclarations(&deps, &declarations, "peerDependencies", pkg.PeerDependencies, core.Runtime)

	return &core.Result{
		Name:         pkg.Name,
		Version:      pkg.Version,
		Licenses:     npmLicenses(pkg.License, pkg.Licenses),
		Scripts:      core.StringScripts(pkg.Scripts),
		Dependencies: deps,
		Declarations: declarations,
	}, nil
}

// collectNpmDeclarations appends dependencies and source declarations from
// one package.json dependency block.
func collectNpmDeclarations(
	dependencies *[]core.Dependency,
	declarations *[]core.Declaration,
	location string,
	values map[string]any,
	scope core.Scope,
) {
	for name, value := range values {
		if isNpmComment(name) {
			continue
		}
		version, ok := value.(string)
		if !ok {
			continue
		}
		realName, realVersion := parseNpmAlias(name, version)
		*dependencies = append(*dependencies, core.Dependency{
			Name:    realName,
			Version: realVersion,
			Scope:   scope,
			Direct:  true,
		})
		*declarations = append(*declarations, core.Declaration{
			Name:     realName,
			Version:  realVersion,
			Scope:    scope,
			Direct:   true,
			Location: location + "/" + url.PathEscape(name),
		})
	}
}

func npmLicenses(license any, legacy []npmLicense) []string {
	var licenses []string
	switch value := license.(type) {
	case string:
		if value != "" {
			licenses = append(licenses, value)
		}
	case map[string]any:
		if valueType, ok := value["type"].(string); ok && valueType != "" {
			licenses = append(licenses, valueType)
		}
	}
	for _, value := range legacy {
		if value.Type != "" {
			licenses = append(licenses, value.Type)
		}
	}
	return licenses
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

type packageLockRoot struct {
	Dependencies         map[string]json.RawMessage `json:"dependencies"`
	DevDependencies      map[string]json.RawMessage `json:"devDependencies"`
	OptionalDependencies map[string]json.RawMessage `json:"optionalDependencies"`
	PeerDependencies     map[string]json.RawMessage `json:"peerDependencies"`
}

const packageLockPackagesField = "packages"

var packageLockPackagesKey = []byte(`"` + packageLockPackagesField + `"`)

var errPackageLock = errors.New("malformed JSON")

func (p *npmPackageLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	// Without a packages section the lockfile is v1: nested dependencies only.
	if !bytes.Contains(content, packageLockPackagesKey) || !hasPackageLockPackages(content) {
		var lock packageLockJSON
		if err := json.Unmarshal(content, &lock); err != nil {
			return nil, &core.ParseError{Filename: filename, Err: err}
		}
		return &core.Result{Dependencies: parsePackageLockV1(lock.Dependencies)}, nil
	}

	// npm writes one key per line, which the line scanner reads without
	// decoding the document.
	if deps := parsePackageLockV3Lines(content); len(deps) > 0 {
		return &core.Result{Dependencies: deps}, nil
	}

	// Any other formatting, compact JSON included.
	deps, err := decodePackageLock(content)
	if err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
	return &core.Result{Dependencies: deps}, nil
}

func hasPackageLockPackages(content []byte) bool {
	decoder := newPackageLockDecoder(content)
	if !decodeJSONObjectOpening(decoder) {
		return false
	}
	for decoder.PeekKind() != '}' {
		key, ok := decodeJSONKey(decoder)
		if !ok {
			return false
		}
		if key == packageLockPackagesField {
			return true
		}
		if !skipJSONValue(decoder) {
			return false
		}
	}
	return false
}

func parsePackageLockV1(deps map[string]packageLockDep) []core.Dependency {
	count := countPackageLockV1(deps)
	if count == 0 {
		return nil
	}
	return appendPackageLockV1(make([]core.Dependency, 0, count), deps)
}

func countPackageLockV1(deps map[string]packageLockDep) int {
	count := len(deps)
	for _, dep := range deps {
		count += countPackageLockV1(dep.Dependencies)
	}
	return count
}

func appendPackageLockV1(result []core.Dependency, deps map[string]packageLockDep) []core.Dependency {
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

		if len(dep.Dependencies) > 0 {
			result = appendPackageLockV1(result, dep.Dependencies)
		}
	}
	return result
}

// packageLockEntry is one entry of the v2/v3 "packages" object, filled either
// by the line scanner or by the JSON decoder. Path holds the install path the
// entry is keyed by.
type packageLockEntry struct {
	Path        string `json:"-"`
	Version     string `json:"version"`
	Integrity   string `json:"integrity"`
	Resolved    string `json:"resolved"`
	Dev         bool   `json:"dev"`
	Optional    bool   `json:"optional"`
	DevOptional bool   `json:"devOptional"`
	Link        bool   `json:"link"`
}

func (e *packageLockEntry) reset(path string) {
	*e = packageLockEntry{Path: path}
}

func (e *packageLockEntry) hasContent() bool {
	return e.Path != "" && (e.Version != "" || e.Link)
}

func (e *packageLockEntry) toDependency(directDependencies map[string]bool) (core.Dependency, bool) {
	name := extractPackageName(e.Path)
	if name == "" || (e.Version == "" && !e.Link) {
		return core.Dependency{}, false
	}
	scope := core.Runtime
	if e.Dev || e.DevOptional {
		scope = core.Development
	} else if e.Optional {
		scope = core.Optional
	}
	topLevel := !strings.Contains(strings.TrimPrefix(e.Path, "node_modules/"), "node_modules/")
	direct := topLevel && directDependencies[name]
	return core.Dependency{
		Name:        name,
		Version:     e.Version,
		Scope:       scope,
		Integrity:   e.Integrity,
		Direct:      direct,
		RegistryURL: e.Resolved,
	}, true
}

func parsePackageLockDirectDependencies(content []byte) map[string]bool {
	decoder := newPackageLockDecoder(content)
	if !decodeJSONObjectOpening(decoder) {
		return nil
	}

	for decoder.PeekKind() != '}' {
		key, ok := decodeJSONKey(decoder)
		if !ok {
			return nil
		}
		if key == packageLockPackagesField {
			return decodePackageLockRootDependencies(decoder)
		}
		if !skipJSONValue(decoder) {
			return nil
		}
	}

	return nil
}

func decodePackageLockRootDependencies(decoder *jsontext.Decoder) map[string]bool {
	if !decodeJSONObjectOpening(decoder) {
		return nil
	}

	for decoder.PeekKind() != '}' {
		path, ok := decodeJSONKey(decoder)
		if !ok {
			return nil
		}
		if path == "" {
			declared, err := decodePackageLockRoot(decoder)
			if err != nil {
				return nil
			}
			return declared
		}
		if !skipJSONValue(decoder) {
			return nil
		}
	}

	return nil
}

func decodePackageLockRoot(decoder *jsontext.Decoder) (map[string]bool, error) {
	var root packageLockRoot
	if err := jsonv2.UnmarshalDecode(decoder, &root, json.DefaultOptionsV1()); err != nil {
		return nil, err
	}

	directDependencies := make(map[string]bool)
	collectPackageLockDependencyNames(directDependencies, root.Dependencies)
	collectPackageLockDependencyNames(directDependencies, root.DevDependencies)
	collectPackageLockDependencyNames(directDependencies, root.OptionalDependencies)
	collectPackageLockDependencyNames(directDependencies, root.PeerDependencies)
	return directDependencies, nil
}

func newPackageLockDecoder(content []byte) *jsontext.Decoder {
	return jsontext.NewDecoder(bytes.NewReader(content), json.DefaultOptionsV1())
}

func decodeJSONObjectOpening(decoder *jsontext.Decoder) bool {
	opening, err := decoder.ReadToken()
	return err == nil && opening.Kind() == '{'
}

func decodeJSONKey(decoder *jsontext.Decoder) (string, bool) {
	token, err := decoder.ReadToken()
	if err != nil {
		return "", false
	}
	return token.String(), token.Kind() == '"'
}

func collectPackageLockDependencyNames(directDependencies map[string]bool, dependencies map[string]json.RawMessage) {
	for name := range dependencies {
		directDependencies[name] = true
	}
}

// updateFromLine reads a trimmed line and updates the entry's fields.
// Returns true if the line was consumed.
func (e *packageLockEntry) updateFromLine(trimmed string) bool {
	switch {
	case strings.HasPrefix(trimmed, `"version"`):
		if v := extractJSONStringValue(trimmed); v != "" {
			e.Version = v
		}
	case strings.HasPrefix(trimmed, `"integrity"`):
		if v := extractJSONStringValue(trimmed); v != "" {
			e.Integrity = v
		}
	case strings.HasPrefix(trimmed, `"resolved"`):
		if v := extractJSONStringValue(trimmed); v != "" {
			e.Resolved = v
		}
	case strings.HasPrefix(trimmed, `"dev": true`):
		e.Dev = true
	case strings.HasPrefix(trimmed, `"optional": true`):
		e.Optional = true
	case strings.HasPrefix(trimmed, `"devOptional": true`):
		e.DevOptional = true
	case strings.HasPrefix(trimmed, `"link": true`):
		e.Link = true
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
	line = strings.TrimSuffix(line, "\r")
	return (line == "  }," || line == "  }") && strings.HasPrefix(trimmed, "}")
}

// parsePackageLockV3Lines parses v3 format using line-based parsing.
// Format: "packages": { "node_modules/name": { "version": "x", ... } }
func parsePackageLockV3Lines(content []byte) []core.Dependency {
	text := string(content)
	count := 0
	for range packageLockV3Entries(text) {
		count++
	}
	var deps []core.Dependency
	if count > 0 {
		deps = make([]core.Dependency, 0, count)
	}
	directDependencies := parsePackageLockDirectDependencies(content)
	for entry := range packageLockV3Entries(text) {
		if dep, ok := entry.toDependency(directDependencies); ok {
			deps = append(deps, dep)
		}
	}
	if len(deps) == 0 {
		return nil
	}
	return deps
}

func packageLockV3Entries(content string) iter.Seq[packageLockEntry] {
	return func(yield func(packageLockEntry) bool) {
		inPackages := false
		var entry packageLockEntry

		for line := range strings.SplitSeq(content, "\n") {
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
				if entry.hasContent() && !yield(entry) {
					return
				}
				entry.reset(extractQuotedPath(trimmed))
				continue
			}

			entry.updateFromLine(trimmed)
		}

		if entry.hasContent() {
			yield(entry)
		}
	}
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

// decodePackageLock decodes the document, for lockfiles the line scanner cannot
// read. v2 and v3 list every installed package under "packages"; v1 only has
// the nested "dependencies" tree.
func decodePackageLock(content []byte) ([]core.Dependency, error) {
	decoder := newPackageLockDecoder(content)
	if !decodeJSONObjectOpening(decoder) {
		return nil, errPackageLock
	}

	var tree map[string]packageLockDep
	for decoder.PeekKind() != '}' {
		key, ok := decodeJSONKey(decoder)
		if !ok {
			return nil, errPackageLock
		}
		switch key {
		case packageLockPackagesField:
			if !decodeJSONObjectOpening(decoder) {
				return nil, errPackageLock
			}
			return decodePackageLockEntries(decoder)
		case "dependencies":
			if err := jsonv2.UnmarshalDecode(decoder, &tree, json.DefaultOptionsV1()); err != nil {
				return nil, err
			}
		default:
			if !skipJSONValue(decoder) {
				return nil, errPackageLock
			}
		}
	}

	return parsePackageLockV1(tree), nil
}

// decodePackageLockEntries reads the entries of an open "packages" object in
// document order. The root entry is converted last because it can appear after
// the packages it declares.
func decodePackageLockEntries(decoder *jsontext.Decoder) ([]core.Dependency, error) {
	var entries []packageLockEntry
	var directDependencies map[string]bool

	for decoder.PeekKind() != '}' {
		path, ok := decodeJSONKey(decoder)
		if !ok {
			return nil, errPackageLock
		}
		if path == "" {
			declared, err := decodePackageLockRoot(decoder)
			if err != nil {
				return nil, err
			}
			directDependencies = declared
			continue
		}
		entries = append(entries, packageLockEntry{Path: path})
		if err := jsonv2.UnmarshalDecode(decoder, &entries[len(entries)-1], json.DefaultOptionsV1()); err != nil {
			return nil, err
		}
	}

	deps := make([]core.Dependency, 0, len(entries))
	for i := range entries {
		if dep, ok := entries[i].toDependency(directDependencies); ok {
			deps = append(deps, dep)
		}
	}
	if len(deps) == 0 {
		return nil, nil
	}
	return deps, nil
}

// skipJSONValue reads past the next value without copying it.
func skipJSONValue(decoder *jsontext.Decoder) bool {
	return decoder.SkipValue() == nil
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
	Version      string              `json:"version"`
	Resolved     string              `json:"resolved"`
	Integrity    string              `json:"integrity"`
	Dev          bool                `json:"dev"`
	Dependencies map[string]npmLsDep `json:"dependencies"`
}

func (p *npmLsParser) Parse(filename string, content []byte) (*core.Result, error) {
	var ls npmLsJSON
	if err := json.Unmarshal(content, &ls); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	return &core.Result{Dependencies: parseNpmLsDeps(ls.Dependencies, make(map[string]bool))}, nil
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
