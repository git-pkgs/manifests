package maven

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("maven", core.Manifest, &sbtParser{}, core.ExactMatch("build.sbt"))

	// dependencies-*.dot - lockfile (sbt dependencyDot output)
	core.Register("maven", core.Lockfile, &sbtDotParser{}, sbtDotMatcher)
}

// sbtParser parses build.sbt files.
type sbtParser struct{}

var (
	// Match: libraryDependencies += "group" % "artifact" % "version"
	// or: libraryDependencies += "group" %% "artifact" % "version"
	sbtDepRegex = regexp.MustCompile(`"([^"]+)"\s*%%?\s*"([^"]+)"\s*%\s*"([^"]+)"`)
	// Match: "group" % "artifact" % "version" % "scope"
	sbtDepWithScopeRegex = regexp.MustCompile(`"([^"]+)"\s*%%?\s*"([^"]+)"\s*%\s*"([^"]+)"\s*%\s*"([^"]+)"`)
)

func (p *sbtParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	// Parse deps with scope first
	for _, match := range sbtDepWithScopeRegex.FindAllStringSubmatch(text, -1) {
		group := match[1]
		artifact := match[2]
		version := match[3]
		scopeStr := strings.ToLower(match[4])

		name := group + ":" + artifact
		if seen[name] {
			continue
		}
		seen[name] = true

		var scope core.Scope
		switch scopeStr {
		case "test":
			scope = core.Test
		case "provided":
			scope = core.Optional
		default:
			scope = core.Runtime
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   scope,
			Direct:  true,
		})
	}

	// Parse deps without scope
	for _, match := range sbtDepRegex.FindAllStringSubmatch(text, -1) {
		group := match[1]
		artifact := match[2]
		version := match[3]

		name := group + ":" + artifact
		if seen[name] {
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

	return deps, nil
}

// sbtDotMatcher matches sbt dependency DOT files (e.g., dependencies-compile.dot)
func sbtDotMatcher(filename string) bool {
	return strings.HasPrefix(filename, "dependencies-") && strings.HasSuffix(filename, ".dot")
}

// sbtDotParser parses sbt dependencyDot output files.
type sbtDotParser struct{}

// Match "group:artifact:version" in DOT node/edge definitions
var sbtDotDepRegex = regexp.MustCompile(`"([^":]+):([^":]+):([^"]+)"`)

func (p *sbtDotParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[string]bool)
	text := string(content)

	// Determine scope from filename
	scope := core.Runtime
	if strings.Contains(filename, "-test") {
		scope = core.Test
	}

	// Find all group:artifact:version patterns
	for _, match := range sbtDotDepRegex.FindAllStringSubmatch(text, -1) {
		group := match[1]
		artifact := match[2]
		version := match[3]

		name := group + ":" + artifact
		if seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   scope,
			Direct:  false, // DOT file shows full graph, not just direct deps
		})
	}

	return deps, nil
}
