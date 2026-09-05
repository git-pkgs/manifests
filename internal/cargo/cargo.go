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
	}

	if _, err := toml.Decode(string(content), &cargo); err != nil {
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
	return &core.Result{
		Name:         pkgName,
		Version:      string(cargo.Package.Version),
		Licenses:     licenses,
		LicenseFile:  string(cargo.Package.LicenseFile),
		Dependencies: filtered,
		Declarations: declarations,
	}, nil
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
