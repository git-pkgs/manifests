package alpine

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("alpine", core.Manifest, &apkbuildParser{}, core.ExactMatch("APKBUILD"))
}

// apkbuildParser parses Alpine Linux APKBUILD files.
type apkbuildParser struct{}

var (
	// Matches variable assignments like: depends="foo bar"
	apkVarRegex = regexp.MustCompile(`^(\w+)="([^"]*)"`)
	// Matches multi-line variable start: depends="
	apkVarStartRegex = regexp.MustCompile(`^(\w+)="([^"]*)$`)
	// Matches package with optional version: pkg>=1.0 or pkg
	apkDepRegex = regexp.MustCompile(`^([a-zA-Z0-9_][a-zA-Z0-9_+.-]*)(>=|<=|>|<|=)?(.*)$`)
)

func (p *apkbuildParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	vars := parseApkbuildVars(string(content))

	var deps []core.Dependency

	// Runtime dependencies
	for _, dep := range parseApkDeps(vars["depends"]) {
		dep.Scope = core.Runtime
		dep.Direct = true
		deps = append(deps, dep)
	}

	// Development dependencies
	for _, dep := range parseApkDeps(vars["depends_dev"]) {
		dep.Scope = core.Development
		dep.Direct = true
		deps = append(deps, dep)
	}

	// Build dependencies
	for _, dep := range parseApkDeps(vars["makedepends"]) {
		dep.Scope = core.Build
		dep.Direct = true
		deps = append(deps, dep)
	}

	// Test/check dependencies
	for _, dep := range parseApkDeps(vars["checkdepends"]) {
		dep.Scope = core.Test
		dep.Direct = true
		deps = append(deps, dep)
	}

	return deps, nil
}

func parseApkbuildVars(content string) map[string]string {
	vars := make(map[string]string)
	lines := strings.Split(content, "\n")

	var currentVar string
	var currentValue strings.Builder

	for _, line := range lines {
		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Check if we're in a multi-line value
		if currentVar != "" {
			if strings.HasSuffix(strings.TrimSpace(line), "\"") {
				// End of multi-line
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSuffix(strings.TrimSpace(line), "\""))
				vars[currentVar] = currentValue.String()
				currentVar = ""
				currentValue.Reset()
			} else {
				currentValue.WriteString(" ")
				currentValue.WriteString(strings.TrimSpace(line))
			}
			continue
		}

		// Check for single-line variable
		if match := apkVarRegex.FindStringSubmatch(line); match != nil {
			vars[match[1]] = match[2]
			continue
		}

		// Check for multi-line variable start
		if match := apkVarStartRegex.FindStringSubmatch(line); match != nil {
			currentVar = match[1]
			currentValue.WriteString(match[2])
		}
	}

	return vars
}

func parseApkDeps(depStr string) []core.Dependency {
	var deps []core.Dependency

	// Expand $depends_dev and similar references (simplified - just skip them)
	fields := strings.Fields(depStr)

	for _, field := range fields {
		// Skip variable references
		if strings.HasPrefix(field, "$") {
			continue
		}

		if match := apkDepRegex.FindStringSubmatch(field); match != nil {
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
	}

	return deps
}
