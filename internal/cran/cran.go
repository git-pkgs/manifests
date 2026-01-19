package cran

import (
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

func (p *descriptionParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
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

	return deps, nil
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
	Packages map[string]renvPackage `json:"Packages"`
}

type renvPackage struct {
	Package string `json:"Package"`
	Version string `json:"Version"`
	Hash    string `json:"Hash"`
}

func (p *renvLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock renvLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Packages {
		integrity := ""
		if pkg.Hash != "" {
			integrity = "md5-" + pkg.Hash
		}

		deps = append(deps, core.Dependency{
			Name:      pkg.Package,
			Version:   pkg.Version,
			Scope:   core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return deps, nil
}
