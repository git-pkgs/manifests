package nuget

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"encoding/xml"
	"regexp"
	"strings"
)

func init() {
	core.Register("nuget", core.Manifest, &csprojParser{}, core.SuffixMatch(".csproj"))
	core.Register("nuget", core.Manifest, &csprojParser{}, core.SuffixMatch(".vbproj"))
	core.Register("nuget", core.Manifest, &csprojParser{}, core.SuffixMatch(".fsproj"))
	core.Register("nuget", core.Manifest, &nuspecParser{}, core.SuffixMatch(".nuspec"))
	core.Register("nuget", core.Manifest, &packagesConfigParser{}, core.ExactMatch("packages.config"))
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
	ItemGroups []csprojItemGroup `xml:"ItemGroup"`
}

type csprojItemGroup struct {
	PackageRefs []csprojPackageRef `xml:"PackageReference"`
	References  []csprojReference  `xml:"Reference"`
}

type csprojPackageRef struct {
	Include string `xml:"Include,attr"`
	Version string `xml:"Version,attr"`
	VerElem string `xml:"Version"`
}

type csprojReference struct {
	Include  string `xml:"Include,attr"`
	HintPath string `xml:"HintPath"`
}

func (p *csprojParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var project csprojProject
	if err := xml.Unmarshal(content, &project); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, group := range project.ItemGroups {
		// Parse PackageReference elements
		for _, ref := range group.PackageRefs {
			name := ref.Include
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true

			version := ref.Version
			if version == "" {
				version = strings.TrimSpace(ref.VerElem)
			}

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}

		// Parse Reference elements (legacy format)
		for _, ref := range group.References {
			if ref.Include == "" {
				continue
			}

			// Parse Include attribute: "Name, Version=x.x.x.x, Culture=neutral, ..."
			name, version := parseReferenceInclude(ref.Include)
			if name == "" || seen[name] {
				continue
			}
			// Skip system assemblies
			if isSystemAssembly(name) {
				continue
			}
			seen[name] = true

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	return deps, nil
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
		Dependencies struct {
			Groups []nuspecDepGroup `xml:"group"`
			Deps   []nuspecDep      `xml:"dependency"`
		} `xml:"dependencies"`
	} `xml:"metadata"`
}

type nuspecDepGroup struct {
	TargetFramework string      `xml:"targetFramework,attr"`
	Deps            []nuspecDep `xml:"dependency"`
}

type nuspecDep struct {
	ID      string `xml:"id,attr"`
	Version string `xml:"version,attr"`
}

func (p *nuspecParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var pkg nuspecPackage
	if err := xml.Unmarshal(content, &pkg); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	// Parse ungrouped dependencies
	for _, dep := range pkg.Metadata.Dependencies.Deps {
		if dep.ID == "" || seen[dep.ID] {
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
		for _, dep := range group.Deps {
			if dep.ID == "" || seen[dep.ID] {
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

	return deps, nil
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

func (p *packagesConfigParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var config packagesConfig
	if err := xml.Unmarshal(content, &config); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

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
	}

	return deps, nil
}

// packagesLockParser parses packages.lock.json files.
type packagesLockParser struct{}

type packagesLockJSON struct {
	Version      int                                       `json:"version"`
	Dependencies map[string]map[string]packagesLockPkg `json:"dependencies"`
}

type packagesLockPkg struct {
	Type        string `json:"type"`
	Resolved    string `json:"resolved"`
	ContentHash string `json:"contentHash"`
}

func (p *packagesLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
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
				Name:    name,
				Version: pkg.Resolved,
				Scope:   core.Runtime,
				Direct:  direct,
			})
		}
	}

	return deps, nil
}

// paketLockParser parses paket.lock files.
type paketLockParser struct{}

var (
	// Match indented package line: "    PackageName (version)"
	paketPkgRegex = regexp.MustCompile(`^\s{4}([A-Za-z][A-Za-z0-9._-]*)\s+\(([^)]+)\)`)
)

func (p *paketLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
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

	return deps, nil
}

// projectAssetsParser parses project.assets.json files.
type projectAssetsParser struct{}

type projectAssetsJSON struct {
	Targets map[string]map[string]struct {
		Type string `json:"type"`
	} `json:"targets"`
}

func (p *projectAssetsParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
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
			parts := strings.SplitN(key, "/", 2)
			if len(parts) != 2 {
				continue
			}

			name := parts[0]
			version := parts[1]

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

	return deps, nil
}

// projectJSONParser parses Project.json files (legacy DNX/ASP.NET 5 format).
type projectJSONParser struct{}

type projectJSON struct {
	Dependencies map[string]any `json:"dependencies"`
}

func (p *projectJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var proj projectJSON
	if err := json.Unmarshal(content, &proj); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

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
	}

	return deps, nil
}

// depsJSONParser parses *.deps.json files (.NET Core runtime deps).
type depsJSONParser struct{}

type depsJSON struct {
	Libraries map[string]struct {
		Type       string `json:"type"`
		Serviceable bool   `json:"serviceable"`
		SHA512     string `json:"sha512"`
	} `json:"libraries"`
}

func (p *depsJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps depsJSON
	if err := json.Unmarshal(content, &deps); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var result []core.Dependency

	for key, lib := range deps.Libraries {
		// Skip project types
		if lib.Type == "project" {
			continue
		}

		// Key format: "Name/Version"
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		version := parts[1]

		integrity := ""
		if lib.SHA512 != "" {
			integrity = "sha512-" + lib.SHA512
		}

		result = append(result, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:     core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return result, nil
}

// projectLockJSONParser parses Project.lock.json files (legacy DNX format).
type projectLockJSONParser struct{}

type projectLockJSON struct {
	Libraries map[string]struct {
		Type string `json:"type"`
		SHA512 string `json:"sha512"`
	} `json:"libraries"`
}

func (p *projectLockJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock projectLockJSON
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for key, lib := range lock.Libraries {
		// Skip project types
		if lib.Type == "project" {
			continue
		}

		// Key format: "Name/Version"
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			continue
		}

		name := parts[0]
		version := parts[1]

		integrity := ""
		if lib.SHA512 != "" {
			integrity = "sha512-" + lib.SHA512
		}

		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:     core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return deps, nil
}
