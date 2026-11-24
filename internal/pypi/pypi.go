package pypi

import (
	"encoding/json"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/git-pkgs/manifests/internal/core"
)

const (
	groupDev         = "dev"
	groupDevelopment = "development"
	groupTest        = "test"
)

func init() {
	// requirements.txt variants - manifests
	core.Register("pypi", core.Manifest, &requirementsTxtParser{},
		core.AnyMatch(
			core.ExactMatch("requirements.txt", "requirements.frozen"),
			core.GlobMatch("*requirements*.txt"),
			core.GlobMatch("requirements/*.txt"),
		))

	// Pipfile - manifest
	core.Register("pypi", core.Manifest, &pipfileParser{}, core.ExactMatch("Pipfile"))

	// Pipfile.lock - lockfile
	core.Register("pypi", core.Lockfile, &pipfileLockParser{}, core.ExactMatch("Pipfile.lock"))

	// pyproject.toml - manifest (Poetry/PEP 621)
	core.Register("pypi", core.Manifest, &pyprojectParser{}, core.ExactMatch("pyproject.toml"))

	// poetry.lock - lockfile
	core.Register("pypi", core.Lockfile, &poetryLockParser{}, core.ExactMatch("poetry.lock"))

	// pdm.lock - lockfile
	core.Register("pypi", core.Lockfile, &pdmLockParser{}, core.ExactMatch("pdm.lock"))

	// uv.lock - lockfile
	core.Register("pypi", core.Lockfile, &uvLockParser{}, core.ExactMatch("uv.lock"))

	// pip-dependency-graph.json, pipdeptree.json, pipenv.graph.json - lockfile (pipdeptree --json output)
	core.Register("pypi", core.Lockfile, &pipDependencyGraphParser{},
		core.ExactMatch("pip-dependency-graph.json", "pipdeptree.json", "pipenv.graph.json"))

	// pip-resolved-dependencies.txt - lockfile (pip freeze output)
	core.Register("pypi", core.Lockfile, &pipResolvedDepsParser{}, core.ExactMatch("pip-resolved-dependencies.txt"))

	// setup.py - manifest
	core.Register("pypi", core.Manifest, &setupPyParser{}, core.ExactMatch("setup.py"))
	core.Register("pypi", core.Manifest, &setupCfgParser{}, core.ExactMatch("setup.cfg"))

	// pylock.toml - lockfile (PEP 751)
	core.Register("pypi", core.Lockfile, &pylockTomlParser{}, core.ExactMatch("pylock.toml"))
}

// requirementsTxtParser parses requirements.txt files.
type requirementsTxtParser struct{}

func (p *requirementsTxtParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	var declarations []core.Declaration
	locations := make(map[string]int)

	for line := range strings.SplitSeq(string(content), "\n") {
		line = requirementsLine(line)
		// Skip empty lines and options
		if line == "" || strings.HasPrefix(line, "-") {
			continue
		}
		if requirement, ok := parsePEP508Requirement(line); ok {
			appendPEP508(&deps, &declarations, locations, "requirements", requirement, core.Runtime)
		}
	}

	return &core.Result{Dependencies: deps, Declarations: declarations}, nil
}

// requirementsLine returns the requirement on a requirements.txt line without
// its comment, a trailing line continuation or per-requirement options such
// as --hash. As in pip, "#" starts a comment at the start of the line or
// after white space, so a URL fragment is kept.
func requirementsLine(line string) string {
	for index := 0; index < len(line); index++ {
		if line[index] == '#' && (index == 0 || line[index-1] == ' ' || line[index-1] == '\t') {
			line = line[:index]
			break
		}
	}
	line = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\\"))
	for _, separator := range []string{" --", "\t--"} {
		if index := strings.Index(line, separator); index > 0 {
			line = strings.TrimSpace(line[:index])
		}
	}
	return line
}

// pipfileParser parses Pipfile (TOML format).
type pipfileParser struct{}

func (p *pipfileParser) Parse(filename string, content []byte) (*core.Result, error) {
	var pipfile struct {
		Packages    map[string]any `toml:"packages"`
		DevPackages map[string]any `toml:"dev-packages"`
	}

	if _, err := toml.Decode(string(content), &pipfile); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, value := range pipfile.Packages {
		version := extractPipfileVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, value := range pipfile.DevPackages {
		version := extractPipfileVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

func extractPipfileVersion(value any) string {
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

// pipfileLockParser parses Pipfile.lock (JSON format).
type pipfileLockParser struct{}

type pipfileLock struct {
	Meta    pipfileLockMeta           `json:"_meta"`
	Default map[string]pipfileLockDep `json:"default"`
	Develop map[string]pipfileLockDep `json:"develop"`
}

type pipfileLockMeta struct {
	Sources []pipfileLockSource `json:"sources"`
}

type pipfileLockSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type pipfileLockDep struct {
	Version string   `json:"version"`
	Hashes  []string `json:"hashes"`
	Index   string   `json:"index"`
	File    string   `json:"file"`
}

func (p *pipfileLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock pipfileLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	// Build source name to URL map
	sourceURLs := make(map[string]string)
	for _, src := range lock.Meta.Sources {
		sourceURLs[src.Name] = src.URL
	}

	var deps []core.Dependency

	for name, dep := range lock.Default {
		version := strings.TrimPrefix(dep.Version, "==")
		integrity := ""
		if len(dep.Hashes) > 0 {
			// Use first hash, convert to SRI format
			integrity = convertPythonHash(dep.Hashes[0])
		}

		// Determine registry URL: prefer file URL, then lookup by index
		registryURL := dep.File
		if registryURL == "" && dep.Index != "" {
			registryURL = sourceURLs[dep.Index]
		}

		deps = append(deps, core.Dependency{
			Name:        name,
			Version:     version,
			Scope:       core.Runtime,
			Integrity:   integrity,
			Direct:      false, // Pipfile.lock doesn't distinguish
			RegistryURL: registryURL,
		})
	}

	for name, dep := range lock.Develop {
		version := strings.TrimPrefix(dep.Version, "==")
		integrity := ""
		if len(dep.Hashes) > 0 {
			integrity = convertPythonHash(dep.Hashes[0])
		}

		registryURL := dep.File
		if registryURL == "" && dep.Index != "" {
			registryURL = sourceURLs[dep.Index]
		}

		deps = append(deps, core.Dependency{
			Name:        name,
			Version:     version,
			Scope:       core.Development,
			Integrity:   integrity,
			Direct:      false,
			RegistryURL: registryURL,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// convertPythonHash converts a Python hash (sha256:...) to SRI format (sha256-...).
func convertPythonHash(h string) string {
	if strings.HasPrefix(h, "sha256:") {
		return "sha256-" + strings.TrimPrefix(h, "sha256:")
	}
	if strings.HasPrefix(h, "sha512:") {
		return "sha512-" + strings.TrimPrefix(h, "sha512:")
	}
	return h
}

// pyprojectParser parses pyproject.toml (Poetry format).
type pyprojectParser struct{}

func (p *pyprojectParser) Parse(filename string, content []byte) (*core.Result, error) {
	var pyproject struct {
		Tool struct {
			Poetry struct {
				Name            string         `toml:"name"`
				Version         string         `toml:"version"`
				License         string         `toml:"license"`
				Build           any            `toml:"build"`
				Dependencies    map[string]any `toml:"dependencies"`
				DevDependencies map[string]any `toml:"dev-dependencies"`
				Group           map[string]struct {
					Dependencies map[string]any `toml:"dependencies"`
				} `toml:"group"`
			} `toml:"poetry"`
			PDM struct {
				Build struct {
					CustomHook string `toml:"custom-hook"`
				} `toml:"build"`
			} `toml:"pdm"`
		} `toml:"tool"`
		Project struct {
			Name                 string              `toml:"name"`
			Version              string              `toml:"version"`
			License              any                 `toml:"license"`
			LicenseFiles         []string            `toml:"license-files"`
			Classifiers          []string            `toml:"classifiers"`
			Dependencies         []string            `toml:"dependencies"`
			OptionalDependencies map[string][]string `toml:"optional-dependencies"`
		} `toml:"project"`
		BuildSystem struct {
			// Requires is decoded loosely so a malformed value does not fail
			// the parse.
			Requires any `toml:"requires"`
		} `toml:"build-system"`
	}

	metadata, err := toml.Decode(string(content), &pyproject)
	if err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	var declarations []core.Declaration
	locations := make(map[string]int)

	// Poetry format
	for _, name := range sortedStringKeys(pyproject.Tool.Poetry.Dependencies) {
		value := pyproject.Tool.Poetry.Dependencies[name]
		if name == "python" {
			continue
		}
		version := extractPoetryVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
		appendPyPIDeclaration(&declarations, locations, "tool/poetry/dependencies", name, version, core.Runtime)
	}

	for _, name := range sortedStringKeys(pyproject.Tool.Poetry.DevDependencies) {
		value := pyproject.Tool.Poetry.DevDependencies[name]
		version := extractPoetryVersion(value)
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Development,
			Direct:  true,
		})
		appendPyPIDeclaration(&declarations, locations, "tool/poetry/dev-dependencies", name, version, core.Development)
	}

	// Poetry group dependencies
	for _, groupName := range sortedStringKeys(pyproject.Tool.Poetry.Group) {
		group := pyproject.Tool.Poetry.Group[groupName]
		var scope core.Scope
		switch groupName {
		case groupDev, groupDevelopment:
			scope = core.Development
		case groupTest:
			scope = core.Test
		default:
			scope = core.Runtime
		}

		for _, name := range sortedStringKeys(group.Dependencies) {
			value := group.Dependencies[name]
			version := extractPoetryVersion(value)
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
			location := "tool/poetry/group/" + url.PathEscape(groupName) + "/dependencies"
			appendPyPIDeclaration(&declarations, locations, location, name, version, scope)
		}
	}

	// PEP 621 format
	for _, dep := range pyproject.Project.Dependencies {
		if requirement, ok := parsePEP508Requirement(dep); ok {
			appendPEP508(&deps, &declarations, locations, "project/dependencies", requirement, core.Runtime)
		}
	}

	// PEP 621 optional dependencies
	for _, groupName := range sortedStringKeys(pyproject.Project.OptionalDependencies) {
		groupDeps := pyproject.Project.OptionalDependencies[groupName]
		scope := optionalGroupScope(groupName)
		location := "project/optional-dependencies/" + url.PathEscape(groupName)
		for _, dep := range groupDeps {
			if requirement, ok := parsePEP508Requirement(dep); ok {
				appendPEP508(&deps, &declarations, locations, location, requirement, scope)
			}
		}
	}

	// PEP 518 build requirements. The build-backend is a module path, not a
	// package, so it is not recorded.
	if requires, ok := pyproject.BuildSystem.Requires.([]any); ok {
		for _, value := range requires {
			text, ok := value.(string)
			if !ok {
				continue
			}
			if requirement, ok := parsePEP508Requirement(text); ok {
				appendPEP508(&deps, &declarations, locations, "build-system/requires", requirement, core.Build)
			}
		}
	}

	// PEP 621 [project] takes precedence; fall back to [tool.poetry].
	selfName := pyproject.Project.Name
	if selfName == "" {
		selfName = pyproject.Tool.Poetry.Name
	}
	selfVersion := pyproject.Project.Version
	if selfVersion == "" {
		selfVersion = pyproject.Tool.Poetry.Version
	}

	licenses, licenseFile := pyprojectLicenses(pyproject.Project.License, pyproject.Project.LicenseFiles, pyproject.Project.Classifiers)
	projectDeclaresLicense := metadata.IsDefined("project", "license") ||
		metadata.IsDefined("project", "license-files") || len(licenses) > 0
	if !projectDeclaresLicense && pyproject.Tool.Poetry.License != "" {
		licenses = []string{pyproject.Tool.Poetry.License}
	}
	var scripts map[string][]string
	switch build := pyproject.Tool.Poetry.Build.(type) {
	case string:
		core.AddScript(&scripts, "tool.poetry.build", build)
	case map[string]any:
		if script, ok := build["script"].(string); ok {
			core.AddScript(&scripts, "tool.poetry.build", script)
		}
	}
	core.AddScript(&scripts, "tool.pdm.build.custom-hook", pyproject.Tool.PDM.Build.CustomHook)

	return &core.Result{
		Name:         selfName,
		Version:      selfVersion,
		Licenses:     licenses,
		LicenseFile:  licenseFile,
		Scripts:      scripts,
		Dependencies: deps,
		Declarations: declarations,
	}, nil
}

var pypiNameSeparator = regexp.MustCompile(`[-_.]+`)

// sortedStringKeys returns the keys of values in lexical order.
func sortedStringKeys[V any](values map[string]V) []string {
	return slices.Sorted(maps.Keys(values))
}

// appendPyPIDeclaration records a declaration at a PEP 503-normalized logical
// location and adds a numeric suffix when that location repeats.
func appendPyPIDeclaration(
	declarations *[]core.Declaration,
	locations map[string]int,
	prefix string,
	name string,
	version string,
	scope core.Scope,
) {
	if name == "" {
		return
	}
	identity := pypiNameSeparator.ReplaceAllString(strings.ToLower(name), "-")
	location := core.NextLocation(locations, prefix+"/"+url.PathEscape(identity))
	*declarations = append(*declarations, core.Declaration{
		Name:     name,
		Version:  strings.TrimSpace(version),
		Scope:    scope,
		Direct:   true,
		Location: location,
	})
}

func pyprojectLicenses(license any, licenseFiles, classifiers []string) ([]string, string) {
	var licenses []string
	var licenseFile string
	switch value := license.(type) {
	case string:
		if value != "" {
			licenses = append(licenses, value)
		}
	case map[string]any:
		if text, ok := value["text"].(string); ok && text != "" {
			licenses = append(licenses, text)
		}
		if file, ok := value["file"].(string); ok {
			licenseFile = file
		}
	}
	licenses = append(licenses, licenseClassifiers(classifiers)...)
	if licenseFile == "" && len(licenseFiles) > 0 {
		licenseFile = licenseFiles[0]
	}
	return licenses, licenseFile
}

func licenseClassifiers(classifiers []string) []string {
	var licenses []string
	for _, classifier := range classifiers {
		if strings.HasPrefix(classifier, "License ::") {
			licenses = append(licenses, classifier)
		}
	}
	return licenses
}

func extractPoetryVersion(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		if ver, ok := v["version"].(string); ok {
			return ver
		}
	case []any:
		// Multiple version constraints
		if len(v) > 0 {
			if m, ok := v[0].(map[string]any); ok {
				if ver, ok := m["version"].(string); ok {
					return ver
				}
			}
		}
	}
	return "*"
}

func optionalGroupScope(groupName string) core.Scope {
	switch strings.ToLower(groupName) {
	case groupDev, groupDevelopment, "develop", "lint":
		return core.Development
	case groupTest, "testing", "tests":
		return core.Test
	default:
		return core.Optional
	}
}

// poetryLockParser parses poetry.lock files.
type poetryLockParser struct{}

type poetryLockFile struct {
	Package []poetryLockPackage `toml:"package"`
}

type poetryLockPackage struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Groups  []string `toml:"groups"`
	Files   []struct {
		File string `toml:"file"`
		Hash string `toml:"hash"`
	} `toml:"files"`
	Source struct {
		Type string `toml:"type"`
		URL  string `toml:"url"`
	} `toml:"source"`
}

func (p *poetryLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock poetryLockFile
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Package {
		scope := core.Runtime

		// Determine scope from groups
		for _, g := range pkg.Groups {
			if g == groupDev || g == groupDevelopment {
				scope = core.Development
				break
			}
			if g == groupTest {
				scope = core.Test
				break
			}
		}

		integrity := ""
		if len(pkg.Files) > 0 {
			integrity = convertPythonHash(pkg.Files[0].Hash)
		}

		deps = append(deps, core.Dependency{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Scope:       scope,
			Integrity:   integrity,
			Direct:      false, // poetry.lock doesn't distinguish direct
			RegistryURL: pkg.Source.URL,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// pdmLockParser parses pdm.lock files.
type pdmLockParser struct{}

type pdmLockFile struct {
	Package []pdmLockPackage `toml:"package"`
}

type pdmLockPackage struct {
	Name    string   `toml:"name"`
	Version string   `toml:"version"`
	Groups  []string `toml:"groups"`
	Files   []struct {
		File string `toml:"file"`
		Hash string `toml:"hash"`
	} `toml:"files"`
}

func (p *pdmLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock pdmLockFile
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Package {
		scope := core.Runtime

		// Check if dev dependency
		for _, g := range pkg.Groups {
			if g == groupDev || g == groupDevelopment {
				scope = core.Development
				break
			}
		}

		integrity := ""
		if len(pkg.Files) > 0 && pkg.Files[0].Hash != "" {
			integrity = convertPythonHash(pkg.Files[0].Hash)
		}

		deps = append(deps, core.Dependency{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Scope:     scope,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// uvLockParser parses uv.lock files.
type uvLockParser struct{}

type uvLockFile struct {
	Package []uvLockPackage `toml:"package"`
}

type uvLockPackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Source  struct {
		Registry string `toml:"registry"`
	} `toml:"source"`
	Sdist struct {
		Hash string `toml:"hash"`
	} `toml:"sdist"`
	Wheels []struct {
		Hash string `toml:"hash"`
	} `toml:"wheels"`
}

func (p *uvLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock uvLockFile
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Package {
		integrity := ""
		// Prefer sdist hash, fall back to first wheel hash
		if pkg.Sdist.Hash != "" {
			integrity = convertPythonHash(pkg.Sdist.Hash)
		} else if len(pkg.Wheels) > 0 && pkg.Wheels[0].Hash != "" {
			integrity = convertPythonHash(pkg.Wheels[0].Hash)
		}

		deps = append(deps, core.Dependency{
			Name:        pkg.Name,
			Version:     pkg.Version,
			Scope:       core.Runtime,
			Integrity:   integrity,
			Direct:      false,
			RegistryURL: pkg.Source.Registry,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// pipDependencyGraphParser parses pip-dependency-graph.json files (pipdeptree --json output).
type pipDependencyGraphParser struct{}

type pipDepGraph struct {
	Package struct {
		Key              string `json:"key"`
		PackageName      string `json:"package_name"`
		InstalledVersion string `json:"installed_version"`
	} `json:"package"`
	Dependencies []struct {
		Key              string `json:"key"`
		PackageName      string `json:"package_name"`
		InstalledVersion string `json:"installed_version"`
	} `json:"dependencies"`
}

func (p *pipDependencyGraphParser) Parse(filename string, content []byte) (*core.Result, error) {
	var graph []pipDepGraph
	if err := json.Unmarshal(content, &graph); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, entry := range graph {
		// Add the main package
		name := entry.Package.PackageName
		if name == "" {
			name = entry.Package.Key
		}
		if name != "" && !seen[name] {
			seen[name] = true
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: entry.Package.InstalledVersion,
				Scope:   core.Runtime,
				Direct:  false,
			})
		}
	}

	return &core.Result{Dependencies: deps}, nil
}

// pipResolvedDepsParser parses pip-resolved-dependencies.txt files (pip freeze output).
type pipResolvedDepsParser struct{}

func (p *pipResolvedDepsParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	lines := strings.SplitSeq(string(content), "\n")

	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}

		// Parse package==version format
		const nameVersionParts = 2
		parts := strings.SplitN(line, "==", nameVersionParts)
		if len(parts) != nameVersionParts {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// setupPyParser parses setup.py files.
type setupPyParser struct{}

var (
	// Match quoted string
	quotedStringRegex = regexp.MustCompile(`['"]([^'"]+)['"]`)
	// Match name= keyword argument in setup()
	setupNameRegex = regexp.MustCompile(`\bname\s*=\s*['"]([^'"]+)['"]`)
	// Match version= keyword argument in setup()
	setupVersionRegex = regexp.MustCompile(`\bversion\s*=\s*['"]([^'"]+)['"]`)
	// Match license= keyword argument in setup()
	setupLicenseRegex = regexp.MustCompile(`\blicense\s*=\s*['"]([^'"]+)['"]`)
	// Match classifiers= and license_files= list arguments in setup()
	setupClassifiersRegex  = regexp.MustCompile(`(?s)\bclassifiers\s*=\s*\[([^\]]*)\]`)
	setupLicenseFilesRegex = regexp.MustCompile(`(?s)\blicense_files\s*=\s*\[([^\]]*)\]`)
)

func (p *setupPyParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	contentStr := string(content)

	var selfName, selfVersion string
	if m := setupNameRegex.FindStringSubmatch(contentStr); m != nil {
		selfName = m[1]
	}
	if m := setupVersionRegex.FindStringSubmatch(contentStr); m != nil {
		selfVersion = m[1]
	}
	var licenses []string
	if m := setupLicenseRegex.FindStringSubmatch(contentStr); m != nil {
		licenses = append(licenses, m[1])
	}
	if m := setupClassifiersRegex.FindStringSubmatch(contentStr); m != nil {
		licenses = append(licenses, licenseClassifiers(quotedStrings(m[1]))...)
	}
	var licenseFile string
	if m := setupLicenseFilesRegex.FindStringSubmatch(contentStr); m != nil {
		files := quotedStrings(m[1])
		if len(files) > 0 {
			licenseFile = files[0]
		}
	}

	if start := setupKeywordValue(contentStr, "install_requires", '['); start >= 0 {
		requirements, _ := pythonStringList(contentStr, start)
		for _, text := range requirements {
			if requirement, ok := parsePEP508Requirement(text); ok {
				deps = append(deps, requirement.dependency(core.Runtime))
			}
		}
	}
	if start := setupKeywordValue(contentStr, "extras_require", '{'); start >= 0 {
		for _, group := range setupExtrasRequire(contentStr, start) {
			scope := optionalGroupScope(group.name)
			for _, text := range group.requirements {
				if requirement, ok := parsePEP508Requirement(text); ok {
					deps = append(deps, requirement.dependency(scope))
				}
			}
		}
	}

	return &core.Result{
		Name:         selfName,
		Version:      selfVersion,
		Licenses:     licenses,
		LicenseFile:  licenseFile,
		Dependencies: deps,
	}, nil
}

func quotedStrings(value string) []string {
	var values []string
	for _, match := range quotedStringRegex.FindAllStringSubmatch(value, -1) {
		if len(match) > 1 {
			values = append(values, match[1])
		}
	}
	return values
}

// setupCfgParser parses setup.cfg files.
type setupCfgParser struct{}

func (p *setupCfgParser) Parse(_ string, content []byte) (*core.Result, error) {
	sections := parseSetupCfgSections(string(content))
	metadata := sections["metadata"]
	options := sections["options"]

	var licenses []string
	if license := strings.TrimSpace(metadata["license"]); license != "" {
		licenses = []string{license}
	}
	licenses = append(licenses, licenseClassifiers(configLines(metadata["classifiers"]))...)

	licenseFile := strings.TrimSpace(metadata["license_file"])
	if licenseFile == "" {
		files := configCommaValues(metadata["license_files"])
		if len(files) > 0 {
			licenseFile = files[0]
		}
	}

	var deps []core.Dependency
	for _, text := range configLines(options["install_requires"]) {
		if requirement, ok := parsePEP508Requirement(text); ok {
			deps = append(deps, requirement.dependency(core.Runtime))
		}
	}
	extras := sections["options.extras_require"]
	for _, group := range sortedStringKeys(extras) {
		for _, text := range configLines(extras[group]) {
			if requirement, ok := parsePEP508Requirement(text); ok {
				deps = append(deps, requirement.dependency(optionalGroupScope(group)))
			}
		}
	}

	return &core.Result{
		Name:         strings.TrimSpace(metadata["name"]),
		Version:      strings.TrimSpace(metadata["version"]),
		Licenses:     licenses,
		LicenseFile:  licenseFile,
		Dependencies: deps,
	}, nil
}

func parseSetupCfgSections(content string) map[string]map[string]string {
	sections := make(map[string]map[string]string)
	var section, key string
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, ";") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			section = strings.ToLower(strings.TrimSpace(trimmed[1 : len(trimmed)-1]))
			if sections[section] == nil {
				sections[section] = make(map[string]string)
			}
			key = ""
			continue
		}
		if section == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			if key != "" {
				sections[section][key] += "\n" + trimmed
			}
			continue
		}
		separator := strings.IndexAny(line, "=:")
		if separator < 0 {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(line[:separator]))
		sections[section][key] = strings.TrimSpace(line[separator+1:])
	}
	return sections
}

func configLines(value string) []string {
	var values []string
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if idx := strings.Index(line, " #"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		if line != "" {
			values = append(values, line)
		}
	}
	return values
}

func configCommaValues(value string) []string {
	var values []string
	for _, line := range configLines(value) {
		for item := range strings.SplitSeq(line, ",") {
			if item = strings.TrimSpace(item); item != "" {
				values = append(values, item)
			}
		}
	}
	return values
}

// pylockTomlParser parses pylock.toml files (PEP 751).
type pylockTomlParser struct{}

type pylockToml struct {
	Packages []pylockPackage `toml:"packages"`
}

type pylockPackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Wheels  []struct {
		Name   string `toml:"name"`
		URL    string `toml:"url"`
		Hashes struct {
			SHA256 string `toml:"sha256"`
		} `toml:"hashes"`
	} `toml:"wheels"`
	Archive struct {
		URL    string `toml:"url"`
		Hashes struct {
			SHA256 string `toml:"sha256"`
		} `toml:"hashes"`
	} `toml:"archive"`
}

func (p *pylockTomlParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock pylockToml
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Packages {
		if pkg.Name == "" {
			continue
		}

		integrity := ""
		// Get hash from first wheel or archive
		if len(pkg.Wheels) > 0 && pkg.Wheels[0].Hashes.SHA256 != "" {
			integrity = "sha256-" + pkg.Wheels[0].Hashes.SHA256
		} else if pkg.Archive.Hashes.SHA256 != "" {
			integrity = "sha256-" + pkg.Archive.Hashes.SHA256
		}

		deps = append(deps, core.Dependency{
			Name:      pkg.Name,
			Version:   pkg.Version,
			Scope:     core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}
