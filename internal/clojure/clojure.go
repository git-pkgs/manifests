package clojure

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("clojars", core.Manifest, &projectCljParser{}, core.ExactMatch("project.clj"))
}

// projectCljParser parses Leiningen project.clj files.
type projectCljParser struct{}

var (
	// Matches [group/artifact "version"] or [artifact "version"]
	// The \[+ handles nested brackets from the outer vector
	cljDepRegex = regexp.MustCompile(`\[+([a-zA-Z0-9_./-]+)\s+"([^"]+)"\]`)
)

func (p *projectCljParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)

	// Find and parse :dependencies section
	depsStart := strings.Index(text, ":dependencies")
	if depsStart >= 0 {
		section := extractCljSection(text[depsStart:])
		for _, match := range cljDepRegex.FindAllStringSubmatch(section, -1) {
			deps = append(deps, core.Dependency{
				Name:    match[1],
				Version: match[2],
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	// Find and parse :plugins section (build dependencies)
	pluginsStart := strings.Index(text, ":plugins")
	if pluginsStart >= 0 {
		section := extractCljSection(text[pluginsStart:])
		for _, match := range cljDepRegex.FindAllStringSubmatch(section, -1) {
			deps = append(deps, core.Dependency{
				Name:    match[1],
				Version: match[2],
				Scope:   core.Build,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// extractCljSection extracts the content of a Clojure vector section.
// Given ":dependencies [[...] [...]]", returns "[[...] [...]]".
func extractCljSection(text string) string {
	// Find the opening bracket
	start := strings.Index(text, "[")
	if start < 0 {
		return ""
	}

	// Count brackets to find matching close
	depth := 0
	for i := start; i < len(text); i++ {
		switch text[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}

// normalizeClojureName converts Clojure dep names to Maven format.
// Clojure uses group/artifact or just artifact (which means group=artifact).
func normalizeClojureName(name string) (group, artifact string) {
	if strings.Contains(name, "/") {
		parts := strings.SplitN(name, "/", 2)
		return parts[0], parts[1]
	}
	return name, name
}
