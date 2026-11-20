package cargo

import (
	"net/url"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	// Cargo.toml - manifest
	core.Register("cargo", core.Manifest, &cargoTomlParser{}, core.ExactMatch("Cargo.toml"))

	// Cargo.lock - lockfile
	core.Register("cargo", core.Lockfile, &cargoLockParser{}, core.ExactMatch("Cargo.lock"))
}

// cargoTomlParser parses Cargo.toml files.
type cargoTomlParser struct{}

// inheritableString decodes a Cargo [package] field that may be a literal
// string or the workspace-inheritance table form {workspace = true}. The
// inherited value is not resolvable from a single file, so the table form
// yields the zero value instead of failing the whole decode.
type inheritableString string

func (s *inheritableString) UnmarshalTOML(v any) error {
	if str, ok := v.(string); ok {
		*s = inheritableString(str)
	}
	return nil
}

func (p *cargoTomlParser) Parse(filename string, content []byte) (*core.Result, error) {
	var cargo struct {
		Package struct {
			Name        string            `toml:"name"`
			Version     inheritableString `toml:"version"`
			License     inheritableString `toml:"license"`
			LicenseFile inheritableString `toml:"license-file"`
			Build       any               `toml:"build"`
		} `toml:"package"`
		Dependencies      map[string]any `toml:"dependencies"`
		DevDependencies   map[string]any `toml:"dev-dependencies"`
		BuildDependencies map[string]any `toml:"build-dependencies"`
		Target            map[string]struct {
			Dependencies      map[string]any `toml:"dependencies"`
			DevDependencies   map[string]any `toml:"dev-dependencies"`
			BuildDependencies map[string]any `toml:"build-dependencies"`
		} `toml:"target"`
		Workspace struct {
			Dependencies map[string]any `toml:"dependencies"`
		} `toml:"workspace"`
		Patch   any `toml:"patch"`
		Replace any `toml:"replace"`
	}

	metadata, err := toml.Decode(string(content), &cargo)
	if err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	pkgName := cargo.Package.Name

	collectCargoDependencies(&deps, cargo.Dependencies, core.Runtime)
	collectCargoDependencies(&deps, cargo.DevDependencies, core.Development)
	collectCargoDependencies(&deps, cargo.BuildDependencies, core.Build)
	collectCargoDeclarations(&declarations, "dependencies", cargo.Dependencies, core.Runtime, pkgName)
	collectCargoDeclarations(&declarations, "dev-dependencies", cargo.DevDependencies, core.Development, pkgName)
	collectCargoDeclarations(&declarations, "build-dependencies", cargo.BuildDependencies, core.Build, pkgName)
	for target, groups := range cargo.Target {
		prefix := "target/" + url.PathEscape(target) + "/"
		collectCargoDeclarations(&declarations, prefix+"dependencies", groups.Dependencies, core.Runtime, pkgName)
		collectCargoDeclarations(&declarations, prefix+"dev-dependencies", groups.DevDependencies, core.Development, pkgName)
		collectCargoDeclarations(&declarations, prefix+"build-dependencies", groups.BuildDependencies, core.Build, pkgName)
	}
	collectCargoDeclarations(&declarations, "workspace/dependencies", cargo.Workspace.Dependencies, core.Runtime, pkgName)

	// Filter out self-reference
	filtered := deps[:0]
	for _, d := range deps {
		if d.Name != pkgName {
			filtered = append(filtered, d)
		}
	}

	var licenses []string
	if cargo.Package.License != "" {
		licenses = []string{string(cargo.Package.License)}
	}
	var scripts map[string][]string
	switch build := cargo.Package.Build.(type) {
	case string:
		core.AddScript(&scripts, "build", build)
	case bool:
		if build {
			core.AddScript(&scripts, "build", "build.rs")
		}
	}
	return &core.Result{
		Name:         pkgName,
		Version:      string(cargo.Package.Version),
		Licenses:     licenses,
		LicenseFile:  string(cargo.Package.LicenseFile),
		Scripts:      scripts,
		Dependencies: filtered,
		Declarations: declarations,
		Replacements: cargoReplacements(metadata.Keys(), cargo.Patch, cargo.Replace),
	}, nil
}

// cargoReplacements records [patch.<source>] and [replace] entries in source
// order, without applying them.
func cargoReplacements(keys []toml.Key, patch, replace any) []core.Replacement {
	patches, _ := patch.(map[string]any)
	replaces, _ := replace.(map[string]any)
	if len(patches) == 0 && len(replaces) == 0 {
		return nil
	}
	var replacements []core.Replacement
	seen := make(map[string]bool)
	for _, key := range keys {
		var replacement core.Replacement
		var value any
		switch {
		case len(key) >= 3 && key[0] == "patch":
			registry, _ := patches[key[1]].(map[string]any)
			value = registry[key[2]]
			replacement.Name = key[2]
			replacement.Location = "patch/" + url.PathEscape(key[1]) + "/" + url.PathEscape(key[2])
			if properties, ok := value.(map[string]any); ok {
				if name, ok := properties["package"].(string); ok && name != "" {
					replacement.Name = name
				}
			}
		case len(key) >= 2 && key[0] == "replace":
			value = replaces[key[1]]
			replacement.Name, replacement.Version = cargoPackageIDSpec(key[1])
			replacement.Location = "replace/" + url.PathEscape(key[1])
		default:
			continue
		}
		if !isCargoReplacementValue(value) || seen[replacement.Location] {
			continue
		}
		seen[replacement.Location] = true
		replacement.Source = cargoReplacementSource(value)
		if properties, ok := value.(map[string]any); ok {
			replacement.ReplacementVersion, _ = properties["version"].(string)
		} else if version, ok := value.(string); ok {
			replacement.ReplacementVersion = version
		}
		replacements = append(replacements, replacement)
	}
	return replacements
}

// isCargoReplacementValue reports whether a patch or replace value is a
// dependency table or a version string, the forms Cargo accepts.
func isCargoReplacementValue(value any) bool {
	switch value.(type) {
	case map[string]any, string:
		return true
	}
	return false
}

// cargoPackageIDSpec splits a package ID spec such as "foo:0.1.0",
// "foo@0.1.0" or "https://github.com/rust-lang/crates.io-index#foo@0.1.0".
// A URL with no name in its fragment, as in "https://github.com/x/foo#1.0.0",
// names the package by its last path segment.
func cargoPackageIDSpec(spec string) (string, string) {
	location, fragment, hasFragment := strings.Cut(spec, "#")
	if !hasFragment && !strings.Contains(spec, "://") {
		fragment = spec
	}
	if index := strings.IndexAny(fragment, ":@"); index >= 0 {
		return fragment[:index], fragment[index+1:]
	}
	if fragment != "" && (fragment[0] < '0' || fragment[0] > '9') {
		return fragment, ""
	}
	location, _, _ = strings.Cut(location, "?")
	location = strings.TrimRight(location, "/")
	return location[strings.LastIndex(location, "/")+1:], fragment
}

// cargoReplacementSource returns the git, path or registry source of a patch
// or replace entry.
func cargoReplacementSource(value any) core.Source {
	if properties, ok := value.(map[string]any); ok {
		if path, ok := properties["path"].(string); ok {
			return core.Source{Kind: core.SourcePath, Value: path}
		}
	}
	return extractCargoSource(value)
}

// collectCargoDependencies appends the dependency inventory from one Cargo
// dependency table.
func collectCargoDependencies(dependencies *[]core.Dependency, values map[string]any, scope core.Scope) {
	for name, value := range values {
		if isLocalCargoDep(value) {
			continue
		}
		*dependencies = append(*dependencies, core.Dependency{
			Name:    name,
			Version: extractCargoVersion(value),
			Scope:   scope,
			Direct:  true,
			Source:  extractCargoSource(value),
		})
	}
}

// collectCargoDeclarations appends source declarations from one Cargo
// dependency table.
func collectCargoDeclarations(
	declarations *[]core.Declaration,
	prefix string,
	values map[string]any,
	scope core.Scope,
	selfName string,
) {
	for name, value := range values {
		version := extractCargoVersion(value)
		declaredName, ok := cargoRegistryDeclaration(name, value)
		if !ok || declaredName == selfName {
			continue
		}
		*declarations = append(*declarations, core.Declaration{
			Name:     declaredName,
			Version:  version,
			Scope:    scope,
			Direct:   true,
			Location: prefix + "/" + url.PathEscape(name),
		})
	}
}

// cargoRegistryDeclaration returns the registry package name for a dependency.
// Local, git, workspace and named-registry sources are not registry-checkable.
func cargoRegistryDeclaration(name string, value any) (string, bool) {
	properties, ok := value.(map[string]any)
	if !ok {
		return name, true
	}
	for _, source := range []string{"git", "path", "registry", "workspace"} {
		if _, found := properties[source]; found {
			return "", false
		}
	}
	if packageName, ok := properties["package"].(string); ok && packageName != "" {
		return packageName, true
	}
	return name, true
}

// extractCargoSource returns the explicit source override for a dependency
// table value, or the zero Source for plain and workspace-inherited entries.
func extractCargoSource(value any) core.Source {
	properties, ok := value.(map[string]any)
	if !ok {
		return core.Source{}
	}
	if repo, ok := properties["git"].(string); ok {
		source := core.Source{Kind: core.SourceGit, Value: repo}
		if branch, ok := properties["branch"].(string); ok {
			source.Branch = branch
		}
		if tag, ok := properties["tag"].(string); ok {
			source.Tag = tag
		}
		if rev, ok := properties["rev"].(string); ok {
			source.Ref = rev
		}
		return source
	}
	if registry, ok := properties["registry"].(string); ok {
		return core.Source{Kind: core.SourceRegistry, Value: registry}
	}
	return core.Source{}
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

func (p *cargoLockParser) Parse(filename string, content []byte) (*core.Result, error) {
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
					Name:        currentName,
					Version:     currentVersion,
					Scope:       core.Runtime,
					Integrity:   integrity,
					Direct:      false,
					RegistryURL: extractCargoRegistryURL(currentSource),
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
			Name:        currentName,
			Version:     currentVersion,
			Scope:       core.Runtime,
			Integrity:   integrity,
			Direct:      false,
			RegistryURL: extractCargoRegistryURL(currentSource),
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// extractCargoRegistryURL extracts the registry URL from Cargo's source field.
// Format: "registry+https://github.com/rust-lang/crates.io-index"
func extractCargoRegistryURL(source string) string {
	if strings.HasPrefix(source, "registry+") {
		return strings.TrimPrefix(source, "registry+")
	}
	// For sparse registries: "sparse+https://index.crates.io/"
	if strings.HasPrefix(source, "sparse+") {
		return strings.TrimPrefix(source, "sparse+")
	}
	return source
}
