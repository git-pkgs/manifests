package nuget

import (
	"encoding/json"
	"encoding/xml"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("nuget", core.Manifest, &csprojParser{}, core.SuffixMatch(".csproj"))
	core.Register("nuget", core.Manifest, &csprojParser{}, core.SuffixMatch(".vbproj"))
	core.Register("nuget", core.Manifest, &csprojParser{}, core.SuffixMatch(".fsproj"))
	core.Register("nuget", core.Manifest, &nuspecParser{}, core.SuffixMatch(".nuspec"))
	core.Register("nuget", core.Manifest, &packagesConfigParser{}, core.ExactMatch("packages.config"))
	core.Register("nuget", core.Manifest, &centralPackagesParser{}, core.ExactMatch("Directory.Packages.props"))
	core.Register("nuget", core.Lockfile, &packagesLockParser{}, core.ExactMatch("packages.lock.json"))
	core.Register("nuget", core.Lockfile, &paketLockParser{}, core.ExactMatch("paket.lock"))
	core.Register("nuget", core.Lockfile, &projectAssetsParser{}, core.ExactMatch("project.assets.json"))

	// Project.json - manifest (legacy DNX/ASP.NET 5 format)
	core.Register("nuget", core.Manifest, &projectJSONParser{}, core.ExactMatch("project.json", "Project.json"))

	// *.deps.json - lockfile (.NET Core runtime deps)
	core.Register("nuget", core.Lockfile, &depsJSONParser{}, core.SuffixMatch(".deps.json"))

	// Project.lock.json - lockfile (legacy DNX format)
	core.Register("nuget", core.Lockfile, &projectLockJSONParser{}, core.ExactMatch("project.lock.json", "Project.lock.json"))
}

// csprojParser parses *.csproj, *.vbproj, *.fsproj files.
type csprojParser struct{}

type csprojProject struct {
	PropertyGroups []csprojPropertyGroup `xml:"PropertyGroup"`
	ItemGroups     []csprojItemGroup     `xml:"ItemGroup"`
}

type csprojPropertyGroup struct {
	AssemblyName string `xml:"AssemblyName"`
	PackageID    string `xml:"PackageId"`
	Version      string `xml:"Version"`
}

type csprojItemGroup struct {
	Condition         string                  `xml:"Condition,attr"`
	PackageRefs       []csprojPackageRef      `xml:"PackageReference"`
	PackageVersions   []centralPackageVersion `xml:"PackageVersion"`
	GlobalPackageRefs []centralPackageVersion `xml:"GlobalPackageReference"`
	References        []csprojReference       `xml:"Reference"`
}

type csprojPackageRef struct {
	Include             string `xml:"Include,attr"`
	Update              string `xml:"Update,attr"`
	Condition           string `xml:"Condition,attr"`
	Version             string `xml:"Version,attr"`
	VerElem             string `xml:"Version"`
	VersionOverride     string `xml:"VersionOverride,attr"`
	VersionOverrideElem string `xml:"VersionOverride"`
}

type centralPackageVersion struct {
	Include   string `xml:"Include,attr"`
	Update    string `xml:"Update,attr"`
	Condition string `xml:"Condition,attr"`
	Version   string `xml:"Version,attr"`
	VerElem   string `xml:"Version"`
}

type csprojReference struct {
	Include  string `xml:"Include,attr"`
	HintPath string `xml:"HintPath"`
}

// appendNuGetDeclaration records a NuGet package reference at a
// case-insensitive logical location.
func appendNuGetDeclaration(
	declarations *[]core.Declaration,
	locations map[string]int,
	prefix, name, version string,
	scope core.Scope,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	location := core.NextLocation(locations, prefix+"/"+url.PathEscape(strings.ToLower(name)))
	*declarations = append(*declarations, core.Declaration{
		Name:     name,
		Version:  strings.TrimSpace(version),
		Scope:    scope,
		Direct:   true,
		Location: location,
	})
}

// packageReferenceName returns the Include or Update identity of a project
// package reference.
func packageReferenceName(ref csprojPackageRef) string {
	if ref.Include != "" {
		return ref.Include
	}
	return ref.Update
}

// packageReferenceVersion returns the effective inline version declaration,
// preferring VersionOverride over Version.
func packageReferenceVersion(ref csprojPackageRef) string {
	for _, version := range []string{ref.VersionOverride, ref.VersionOverrideElem, ref.Version, ref.VerElem} {
		if version = strings.TrimSpace(version); version != "" {
			return version
		}
	}
	return ""
}

// packageReferenceDependencyVersion returns the version syntax historically
// exposed through Dependencies for a project package reference.
func packageReferenceDependencyVersion(ref csprojPackageRef) string {
	if ref.Version != "" {
		return ref.Version
	}
	return strings.TrimSpace(ref.VerElem)
}

// collectPackageReferences adds project PackageReference dependencies and
// declarations from one item group.
func collectPackageReferences(
	group csprojItemGroup,
	deps *[]core.Dependency,
	declarations *[]core.Declaration,
	seen map[string]bool,
	locations map[string]int,
) {
	prefix := "package-references"
	if condition := strings.TrimSpace(group.Condition); condition != "" {
		prefix += "/" + url.PathEscape(condition)
	}
	for _, ref := range group.PackageRefs {
		name := packageReferenceName(ref)
		if name == "" {
			continue
		}
		version := packageReferenceVersion(ref)
		refPrefix := prefix
		if condition := strings.TrimSpace(ref.Condition); condition != "" {
			refPrefix += "/" + url.PathEscape(condition)
		}
		appendNuGetDeclaration(declarations, locations, refPrefix, name, version, core.Runtime)
		if ref.Include == "" {
			continue
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		*deps = append(*deps, core.Dependency{
			Name:    name,
			Version: packageReferenceDependencyVersion(ref),
			Scope:   core.Runtime,
			Direct:  true,
		})
	}
}

// collectAssemblyReferences adds legacy project Reference dependencies from
// one item group.
func collectAssemblyReferences(group csprojItemGroup, deps *[]core.Dependency, seen map[string]bool) {
	for _, ref := range group.References {
		if ref.Include == "" {
			continue
		}

		// Parse Include attribute: "Name, Version=x.x.x.x, Culture=neutral, ..."
		name, version := parseReferenceInclude(ref.Include)
		if name == "" || seen[name] || isSystemAssembly(name) {
			continue
		}
		seen[name] = true
		*deps = append(*deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}
}

func (p *csprojParser) Parse(filename string, content []byte) (*core.Result, error) {
	var project csprojProject
	if err := xml.Unmarshal(content, &project); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	seen := make(map[string]bool)
	locations := make(map[string]int)

	for _, group := range project.ItemGroups {
		collectPackageReferences(group, &deps, &declarations, seen, locations)
		collectAssemblyReferences(group, &deps, seen)
	}

	// Default project name is the filename stem; PackageId or AssemblyName
	// override it when present.
	base := filepath.Base(filename)
	selfName := strings.TrimSuffix(base, filepath.Ext(base))
	var selfVersion string
	for _, pg := range project.PropertyGroups {
		if pg.PackageID != "" {
			selfName = pg.PackageID
		} else if pg.AssemblyName != "" {
			selfName = pg.AssemblyName
		}
		if pg.Version != "" {
			selfVersion = pg.Version
		}
	}

	return &core.Result{
		Name:         selfName,
		Version:      selfVersion,
		Dependencies: deps,
		Declarations: declarations,
	}, nil
}

// centralPackagesParser parses Directory.Packages.props files used by NuGet
// central package management.
type centralPackagesParser struct{}

// collectCentralPackageItems records package versions or global package
// references from one central package item group.
func collectCentralPackageItems(
	items []centralPackageVersion,
	groupCondition, prefix string,
	scope core.Scope,
	deps *[]core.Dependency,
	declarations *[]core.Declaration,
	locations map[string]int,
) {
	if condition := strings.TrimSpace(groupCondition); condition != "" {
		prefix += "/" + url.PathEscape(condition)
	}
	for _, pkg := range items {
		name := pkg.Include
		if name == "" {
			name = pkg.Update
		}
		version := pkg.Version
		if version == "" {
			version = strings.TrimSpace(pkg.VerElem)
		}
		if name == "" {
			continue
		}
		if deps != nil {
			*deps = append(*deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
		}
		pkgPrefix := prefix
		if condition := strings.TrimSpace(pkg.Condition); condition != "" {
			pkgPrefix += "/" + url.PathEscape(condition)
		}
		appendNuGetDeclaration(declarations, locations, pkgPrefix, name, version, scope)
	}
}

func (p *centralPackagesParser) Parse(filename string, content []byte) (*core.Result, error) {
	var project csprojProject
	if err := xml.Unmarshal(content, &project); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	locations := make(map[string]int)
	for _, group := range project.ItemGroups {
		collectCentralPackageItems(group.PackageVersions, group.Condition, "package-versions", core.Runtime,
			nil, &declarations, locations)
		collectCentralPackageItems(group.GlobalPackageRefs, group.Condition, "global-package-references", core.Development,
			&deps, &declarations, locations)
	}
	return &core.Result{Dependencies: deps, Declarations: declarations}, nil
}

// parseReferenceInclude parses a Reference Include attribute.
// Format: "Name, Version=x.x.x.x, Culture=neutral, PublicKeyToken=..."
func parseReferenceInclude(include string) (string, string) {
	parts := strings.Split(include, ",")
	name := strings.TrimSpace(parts[0])
	version := ""

	for _, part := range parts[1:] {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "Version=") {
			version = strings.TrimPrefix(part, "Version=")
			break
		}
	}

	return name, version
}

// isSystemAssembly checks if the assembly is a system/framework assembly.
func isSystemAssembly(name string) bool {
	systemPrefixes := []string{
		"System",
		"Microsoft.CSharp",
		"mscorlib",
		"WindowsBase",
		"PresentationCore",
		"PresentationFramework",
	}
	for _, prefix := range systemPrefixes {
		if name == prefix || strings.HasPrefix(name, prefix+".") {
			return true
		}
	}
	return false
}

// nuspecParser parses *.nuspec files.
type nuspecParser struct{}

type nuspecPackage struct {
	Metadata struct {
		ID           string        `xml:"id"`
		Version      string        `xml:"version"`
		License      nuspecLicense `xml:"license"`
		Dependencies struct {
			Groups []nuspecDepGroup `xml:"group"`
			Deps   []nuspecDep      `xml:"dependency"`
		} `xml:"dependencies"`
	} `xml:"metadata"`
}

type nuspecLicense struct {
	Type  string `xml:"type,attr"`
	Value string `xml:",chardata"`
}

type nuspecDepGroup struct {
	TargetFramework string      `xml:"targetFramework,attr"`
	Deps            []nuspecDep `xml:"dependency"`
}

type nuspecDep struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
}

func (p *nuspecParser) Parse(filename string, content []byte) (*core.Result, error) {
	var pkg nuspecPackage
	if err := xml.Unmarshal(content, &pkg); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	seen := make(map[string]bool)
	locations := make(map[string]int)

	// Parse ungrouped dependencies
	for _, dep := range pkg.Metadata.Dependencies.Deps {
		if dep.ID == "" {
			continue
		}
		appendNuGetDeclaration(&declarations, locations, "dependencies", dep.ID, dep.Version, core.Runtime)
		if seen[dep.ID] {
			continue
		}
		seen[dep.ID] = true

		deps = append(deps, core.Dependency{
			Name:    dep.ID,
			Version: dep.Version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	// Parse grouped dependencies
	for _, group := range pkg.Metadata.Dependencies.Groups {
		prefix := "dependency-groups"
		if framework := strings.TrimSpace(group.TargetFramework); framework != "" {
			prefix += "/" + url.PathEscape(framework)
		}
		for _, dep := range group.Deps {
			if dep.ID == "" {
				continue
			}
			appendNuGetDeclaration(&declarations, locations, prefix, dep.ID, dep.Version, core.Runtime)
			if seen[dep.ID] {
				continue
			}
			seen[dep.ID] = true

			deps = append(deps, core.Dependency{
				Name:    dep.ID,
				Version: dep.Version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	result := &core.Result{
		Name:         pkg.Metadata.ID,
		Version:      pkg.Metadata.Version,
		Dependencies: deps,
		Declarations: declarations,
	}
	licenseValue := strings.TrimSpace(pkg.Metadata.License.Value)
	switch strings.ToLower(strings.TrimSpace(pkg.Metadata.License.Type)) {
	case "expression":
		if licenseValue != "" {
			result.Licenses = []string{licenseValue}
		}
	case "file":
		result.LicenseFile = licenseValue
	}
	return result, nil
}

// packagesConfigParser parses packages.config files.
type packagesConfigParser struct{}

type packagesConfig struct {
	Packages []packagesConfigPkg `xml:"package"`
}

type packagesConfigPkg struct {
	ID                    string `xml:"id,attr"`
	Version               string `xml:"version,attr"`
	DevelopmentDependency string `xml:"developmentDependency,attr"`
}

func (p *packagesConfigParser) Parse(filename string, content []byte) (*core.Result, error) {
	var config packagesConfig
	if err := xml.Unmarshal(content, &config); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	locations := make(map[string]int)

	for _, pkg := range config.Packages {
		if pkg.ID == "" {
			continue
		}

		scope := core.Runtime
		if pkg.DevelopmentDependency == "true" {
			scope = core.Development
		}

		deps = append(deps, core.Dependency{
			Name:    pkg.ID,
			Version: pkg.Version,
			Scope:   scope,
			Direct:  true,
		})
		appendNuGetDeclaration(&declarations, locations, "packages", pkg.ID, pkg.Version, scope)
	}

	return &core.Result{Dependencies: deps, Declarations: declarations}, nil
}

// packagesLockParser parses packages.lock.json files.
type packagesLockParser struct{}

type packagesLockJSON struct {
	Version      int                                   `json:"version"`
	Dependencies map[string]map[string]packagesLockPkg `json:"dependencies"`
}

type packagesLockPkg struct {
	Type        string `json:"type"`
	Resolved    string `json:"resolved"`
	ContentHash string `json:"contentHash"`
}

func (p *packagesLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock packagesLockJSON
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, framework := range lock.Dependencies {
		for name, pkg := range framework {
			if seen[name] {
				continue
			}
			seen[name] = true

			direct := pkg.Type == "Direct"

			deps = append(deps, core.Dependency{
				Name:      name,
				Version:   pkg.Resolved,
				Scope:     core.Runtime,
				Integrity: nugetSHA512Integrity(pkg.ContentHash),
				Direct:    direct,
			})
		}
	}

	return &core.Result{Dependencies: deps}, nil
}

// paketLockParser parses paket.lock files.
type paketLockParser struct{}

var (
	// Match indented package line: "    PackageName (version)"
	paketPkgRegex = regexp.MustCompile(`^\s{4}([A-Za-z][A-Za-z0-9._-]*)\s+\(([^)]+)\)`)
)

func (p *paketLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)
	inNuget := false

	for _, line := range lines {
		// Check for NUGET section
		if line == "NUGET" {
			inNuget = true
			continue
		}

		// Check for other top-level sections
		if len(line) > 0 && line[0] != ' ' {
			if line != "NUGET" {
				inNuget = false
			}
			continue
		}

		if !inNuget {
			continue
		}

		// Parse package line
		if match := paketPkgRegex.FindStringSubmatch(line); match != nil {
			name := match[1]
			version := match[2]

			if seen[name] {
				continue
			}
			seen[name] = true

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  false,
			})
		}
	}

	return &core.Result{Dependencies: deps}, nil
}

// projectAssetsParser parses project.assets.json files.
type projectAssetsParser struct{}

type projectAssetsJSON struct {
	Targets map[string]map[string]struct {
		Type string `json:"type"`
	} `json:"targets"`
	Libraries map[string]libraryEntry `json:"libraries"`
}

func (p *projectAssetsParser) Parse(filename string, content []byte) (*core.Result, error) {
	var assets projectAssetsJSON
	if err := json.Unmarshal(content, &assets); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, framework := range assets.Targets {
		for key, pkg := range framework {
			// Skip non-package entries (like "project" types)
			if pkg.Type != "package" {
				continue
			}

			// Key format: "name/version"
			const nameVersionParts = 2
			parts := strings.SplitN(key, "/", nameVersionParts)
			if len(parts) != nameVersionParts {
				continue
			}

			name := parts[0]
			version := parts[1]

			if seen[name] {
				continue
			}
			seen[name] = true

			deps = append(deps, core.Dependency{
				Name:      name,
				Version:   version,
				Scope:     core.Runtime,
				Integrity: nugetSHA512Integrity(assets.Libraries[key].SHA512),
				Direct:    false,
			})
		}
	}

	return &core.Result{Dependencies: deps}, nil
}

// projectJSONParser parses Project.json files (legacy DNX/ASP.NET 5 format).
type projectJSONParser struct{}

type projectJSON struct {
	Dependencies map[string]any `json:"dependencies"`
}

func (p *projectJSONParser) Parse(filename string, content []byte) (*core.Result, error) {
	var proj projectJSON
	if err := json.Unmarshal(content, &proj); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	locations := make(map[string]int)

	for name, value := range proj.Dependencies {
		version := ""
		switch v := value.(type) {
		case string:
			version = v
		case map[string]any:
			if ver, ok := v["version"].(string); ok {
				version = ver
			}
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
		appendNuGetDeclaration(&declarations, locations, "dependencies", name, version, core.Runtime)
	}

	return &core.Result{Dependencies: deps, Declarations: declarations}, nil
}

// libraryEntry holds the fields shared by deps.json and project.lock.json libraries.
type libraryEntry struct {
	Type   string `json:"type"`
	SHA512 string `json:"sha512"`
}

func nugetSHA512Integrity(hash string) string {
	hash = strings.TrimSpace(hash)
	if hash == "" || strings.HasPrefix(hash, "sha512-") {
		return hash
	}

	return "sha512-" + hash
}

// parseLibraries extracts dependencies from a "Name/Version" keyed library map,
// shared by depsJSONParser and projectLockJSONParser.
func parseLibraries(filename string, content []byte) ([]core.Dependency, error) {
	var raw struct {
		Libraries map[string]libraryEntry `json:"libraries"`
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for key, lib := range raw.Libraries {
		if lib.Type == "project" {
			continue
		}

		const nameVersionParts = 2
		parts := strings.SplitN(key, "/", nameVersionParts)
		if len(parts) != nameVersionParts {
			continue
		}

		deps = append(deps, core.Dependency{
			Name:      parts[0],
			Version:   parts[1],
			Scope:     core.Runtime,
			Integrity: nugetSHA512Integrity(lib.SHA512),
			Direct:    false,
		})
	}

	return deps, nil
}

// depsJSONParser parses *.deps.json files (.NET Core runtime deps).
type depsJSONParser struct{}

func (p *depsJSONParser) Parse(filename string, content []byte) (*core.Result, error) {
	deps, err := parseLibraries(filename, content)
	return &core.Result{Dependencies: deps}, err
}

// projectLockJSONParser parses Project.lock.json files (legacy DNX format).
type projectLockJSONParser struct{}

func (p *projectLockJSONParser) Parse(filename string, content []byte) (*core.Result, error) {
	deps, err := parseLibraries(filename, content)
	return &core.Result{Dependencies: deps}, err
}
