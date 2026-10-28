package cran

import (
	"cmp"
	"encoding/json"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("cran", core.Manifest, &descriptionParser{}, core.ExactMatch("DESCRIPTION"))
	core.Register("cran", core.Lockfile, &renvLockParser{}, core.ExactMatch("renv.lock"))
}

// descriptionParser parses R DESCRIPTION files.
type descriptionParser struct{}

func (p *descriptionParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	text := string(content)

	// Parse key-value pairs, handling continuation lines
	fields := parseDescriptionFields(text)

	// Process Depends (runtime)
	if depends, ok := fields["Depends"]; ok {
		for _, dep := range parseRPackageList(depends) {
			if dep.Name == "R" {
				continue // Skip R itself
			}
			dep.Scope = core.Runtime
			dep.Direct = true
			deps = append(deps, dep)
		}
	}

	// Process Imports (runtime)
	if imports, ok := fields["Imports"]; ok {
		for _, dep := range parseRPackageList(imports) {
			dep.Scope = core.Runtime
			dep.Direct = true
			deps = append(deps, dep)
		}
	}

	// Process Suggests (development/optional)
	if suggests, ok := fields["Suggests"]; ok {
		for _, dep := range parseRPackageList(suggests) {
			dep.Scope = core.Development
			dep.Direct = true
			deps = append(deps, dep)
		}
	}

	// Process Enhances (optional)
	if enhances, ok := fields["Enhances"]; ok {
		for _, dep := range parseRPackageList(enhances) {
			dep.Scope = core.Optional
			dep.Direct = true
			deps = append(deps, dep)
		}
	}

	// Process LinkingTo (build)
	if linkingTo, ok := fields["LinkingTo"]; ok {
		for _, dep := range parseRPackageList(linkingTo) {
			dep.Scope = core.Build
			dep.Direct = true
			deps = append(deps, dep)
		}
	}

	licenses, licenseFile := parseRLicense(fields["License"])
	return &core.Result{
		Name:         fields["Package"],
		Version:      fields["Version"],
		Licenses:     licenses,
		LicenseFile:  licenseFile,
		Dependencies: deps,
	}, nil
}

func parseRLicense(value string) ([]string, string) {
	var licenses []string
	var licenseFile string
	for _, alternative := range strings.Split(value, "|") {
		alternative = strings.TrimSpace(alternative)
		if alternative == "" {
			continue
		}
		lower := strings.ToLower(alternative)
		if idx := strings.Index(lower, "file "); idx >= 0 {
			if licenseFile == "" {
				fileFields := strings.Fields(alternative[idx+len("file "):])
				if len(fileFields) > 0 {
					licenseFile = fileFields[0]
				}
			}
			alternative = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(alternative[:idx]), "+"))
		}
		if alternative != "" {
			licenses = append(licenses, alternative)
		}
	}
	return licenses, licenseFile
}

// parseDescriptionFields parses DESCRIPTION file key-value pairs.
func parseDescriptionFields(text string) map[string]string {
	fields := make(map[string]string)
	lines := strings.Split(text, "\n")

	var currentKey string
	var currentValue strings.Builder

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Continuation line (starts with whitespace)
		if line[0] == ' ' || line[0] == '\t' {
			if currentKey != "" {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
			continue
		}

		// Save previous field
		if currentKey != "" {
			fields[currentKey] = currentValue.String()
		}

		// Parse new field
		if idx := strings.Index(line, ":"); idx > 0 {
			currentKey = line[:idx]
			currentValue.Reset()
			currentValue.WriteString(strings.TrimSpace(line[idx+1:]))
		}
	}

	// Save last field
	if currentKey != "" {
		fields[currentKey] = currentValue.String()
	}

	return fields
}

// parseRPackageList parses a comma-separated list of R packages.
func parseRPackageList(list string) []core.Dependency {
	var deps []core.Dependency
	packages := strings.Split(list, ",")

	for _, pkg := range packages {
		pkg = strings.TrimSpace(pkg)
		if pkg == "" {
			continue
		}

		name, version := parseRPackageSpec(pkg)
		if name != "" {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
			})
		}
	}

	return deps
}

// parseRPackageSpec parses a single R package spec like "name" or "name (>= 1.0)".
func parseRPackageSpec(spec string) (name, version string) {
	spec = strings.TrimSpace(spec)

	if idx := strings.Index(spec, "("); idx > 0 {
		name = strings.TrimSpace(spec[:idx])
		// Extract version constraint
		if end := strings.Index(spec, ")"); end > idx {
			version = strings.TrimSpace(spec[idx+1 : end])
		}
	} else {
		name = spec
	}

	return name, version
}

// renvLockParser parses renv.lock files.
type renvLockParser struct{}

type renvLock struct {
	R struct {
		Repositories []renvRepository `json:"Repositories"`
	} `json:"R"`
	Packages map[string]renvPackage `json:"Packages"`
}

type renvRepository struct {
	Name string `json:"Name"`
	URL  string `json:"URL"`
}

type renvPackage struct {
	Package        string `json:"Package"`
	Version        string `json:"Version"`
	Hash           string `json:"Hash"`
	Source         string `json:"Source"`
	Repository     string `json:"Repository"`
	RemoteUsername string `json:"RemoteUsername"`
	RemoteRepo     string `json:"RemoteRepo"`
	RemoteRef      string `json:"RemoteRef"`
	RemoteSha      string `json:"RemoteSha"`
	RemoteSubdir   string `json:"RemoteSubdir"`
	RemoteURL      string `json:"RemoteUrl"`
}

func (p *renvLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock renvLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	repositories := make(map[string]string, len(lock.R.Repositories))
	for _, repo := range lock.R.Repositories {
		if repo.Name != "" && repo.URL != "" {
			repositories[repo.Name] = repo.URL
		}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Packages {
		integrity := ""
		if pkg.Hash != "" {
			integrity = "md5-" + pkg.Hash
		}

		dep := core.Dependency{
			Name:      pkg.Package,
			Version:   pkg.Version,
			Scope:     core.Runtime,
			Integrity: integrity,
			Direct:    false,
		}
		applyRenvSource(&dep, pkg, repositories)
		deps = append(deps, dep)
	}

	return &core.Result{Dependencies: deps}, nil
}

// applyRenvSource sets dep's source and identity from its renv record.
// Entries without a recorded source keep the CRAN identity older lockfiles
// implied.
func applyRenvSource(dep *core.Dependency, pkg renvPackage, repositories map[string]string) {
	ref := cmp.Or(pkg.RemoteSha, pkg.RemoteRef)
	switch pkg.Source {
	case "", "Repository":
		applyRenvRepository(dep, pkg.Repository, repositories)
	case "CRAN":
		applyRenvRepository(dep, cmp.Or(pkg.Repository, pkg.Source), repositories)
	case "Bioconductor":
		dep.Ecosystem = "bioconductor"
		dep.Source = core.Source{Kind: core.SourceRegistry, Value: cmp.Or(pkg.Repository, pkg.Source)}
	case "GitHub":
		dep.NoPURL = true
		dep.Source = core.Source{Kind: core.SourceGitHub, Value: renvRemoteRepo(pkg), Ref: ref, Rel: pkg.RemoteSubdir}
	case "GitLab", "Bitbucket":
		// renv records these by host, owner and name rather than a clone URL,
		// so keep the remotes-style coordinate it accepts for reinstalling.
		value := ""
		if repo := renvRemoteRepo(pkg); repo != "" {
			value = strings.ToLower(pkg.Source) + "::" + repo
		}
		dep.NoPURL = true
		dep.Source = core.Source{Kind: core.SourceGit, Value: value, Ref: ref, Rel: pkg.RemoteSubdir}
	case "git":
		dep.NoPURL = true
		dep.Source = core.Source{Kind: core.SourceGit, Value: pkg.RemoteURL, Ref: ref, Rel: pkg.RemoteSubdir}
	case "URL":
		dep.NoPURL = true
		dep.Source = core.Source{Kind: core.SourceURL, Value: pkg.RemoteURL}
	case "Local":
		dep.NoPURL = true
		dep.Source = core.Source{Kind: core.SourcePath, Value: pkg.RemoteURL}
	default:
		dep.NoPURL = true
	}
}

// applyRenvRepository sets dep's identity from its repository name. A
// repository whose URL is unknown has no package identity.
func applyRenvRepository(dep *core.Dependency, name string, repositories map[string]string) {
	if name == "" {
		return
	}
	dep.Source = core.Source{Kind: core.SourceRegistry, Value: name}
	switch {
	case name == "CRAN" || name == "RSPM" || name == "PPM" || name == "P3M":
		// Posit Package Manager serves CRAN under the RSPM, PPM and P3M names.
	case strings.HasPrefix(strings.ToLower(name), "bioc"):
		dep.Ecosystem = "bioconductor"
	case strings.HasPrefix(name, "https://"), strings.HasPrefix(name, "http://"):
		dep.RegistryURL = name
	case repositories[name] != "":
		dep.RegistryURL = repositories[name]
	default:
		dep.NoPURL = true
	}
}

// renvRemoteRepo returns the owner/name coordinate of a hosted Git remote, or
// empty when either part is missing.
func renvRemoteRepo(pkg renvPackage) string {
	if pkg.RemoteUsername == "" || pkg.RemoteRepo == "" {
		return ""
	}
	return pkg.RemoteUsername + "/" + pkg.RemoteRepo
}
