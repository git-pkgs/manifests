package arch

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("arch", core.Manifest, &pkgbuildParser{}, core.ExactMatch("PKGBUILD"))
}

// pkgbuildParser parses Arch Linux PKGBUILD files.
type pkgbuildParser struct{}

var (
	// Matches array assignments: depends=('foo' 'bar') or depends=("foo" "bar")
	pkgbuildArrayRegex = regexp.MustCompile(`^(\w+)=\(([^)]*)\)`)
	// Matches package with optional version constraint: pkg>=1.0 or pkg
	pkgbuildDepRegex = regexp.MustCompile(`^([a-zA-Z0-9_][a-zA-Z0-9_+@.-]*)(>=|<=|>|<|=)?([^'"]*)$`)
)

func (p *pkgbuildParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	vars := parsePkgbuildVars(string(content))

	var deps []core.Dependency

	// Runtime dependencies
	for _, dep := range parsePkgbuildDeps(vars["depends"]) {
		dep.Scope = core.Runtime
		dep.Direct = true
		deps = append(deps, dep)
	}

	// Build dependencies
	for _, dep := range parsePkgbuildDeps(vars["makedepends"]) {
		dep.Scope = core.Build
		dep.Direct = true
		deps = append(deps, dep)
	}

	// Test/check dependencies
	for _, dep := range parsePkgbuildDeps(vars["checkdepends"]) {
		dep.Scope = core.Test
		dep.Direct = true
		deps = append(deps, dep)
	}

	// Optional dependencies (optdepends)
	for _, dep := range parsePkgbuildDeps(vars["optdepends"]) {
		dep.Scope = core.Optional
		dep.Direct = true
		deps = append(deps, dep)
	}

	return deps, nil
}

func parsePkgbuildVars(content string) map[string]string {
	vars := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		if match := pkgbuildArrayRegex.FindStringSubmatch(line); match != nil {
			vars[match[1]] = match[2]
		}
	}

	return vars
}

func parsePkgbuildDeps(depStr string) []core.Dependency {
	var deps []core.Dependency

	// Extract quoted strings from the array
	// Handles both 'single' and "double" quotes
	var current strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(depStr); i++ {
		c := depStr[i]

		if !inQuote && (c == '\'' || c == '"') {
			inQuote = true
			quoteChar = c
			continue
		}

		if inQuote && c == quoteChar {
			inQuote = false
			// Process the dependency
			depName := current.String()
			current.Reset()

			// Handle optdepends format: "pkg: description"
			if idx := strings.Index(depName, ":"); idx > 0 {
				depName = depName[:idx]
			}

			if match := pkgbuildDepRegex.FindStringSubmatch(depName); match != nil {
				name := match[1]
				version := ""
				if match[2] != "" && match[3] != "" {
					version = match[2] + match[3]
				}

				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
				})
			}
			continue
		}

		if inQuote {
			current.WriteByte(c)
		}
	}

	return deps
}
