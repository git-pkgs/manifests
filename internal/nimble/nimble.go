package nimble

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("nimble", core.Manifest, &nimbleParser{}, core.SuffixMatch(".nimble"))
}

// nimbleParser parses *.nimble files (Nim).
type nimbleParser struct{}

var (
	// Match: requires "pkg >= version" or requires "pkg"
	nimbleRequiresRegex = regexp.MustCompile(`requires\s+"([^"]+)"(?:\s*,\s*"([^"]+)")*`)
	nimbleDepRegex      = regexp.MustCompile(`"([^"]+)"`)
)

func (p *nimbleParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	// Find all requires lines
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "requires") {
			continue
		}

		// Extract all quoted strings from the requires line
		for _, match := range nimbleDepRegex.FindAllStringSubmatch(line, -1) {
			depStr := match[1]
			name, version := parseNimbleDep(depStr)
			if name == "" || seen[name] {
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

// parseNimbleDep parses a nimble dependency string like "nim >= 1.6.0"
func parseNimbleDep(dep string) (name, version string) {
	// Split on comparison operators
	for _, op := range []string{">=", "<=", "==", ">", "<", "~>"} {
		if idx := strings.Index(dep, op); idx > 0 {
			name = strings.TrimSpace(dep[:idx])
			version = op + " " + strings.TrimSpace(dep[idx+len(op):])
			return name, version
		}
	}

	// No version constraint
	return strings.TrimSpace(dep), ""
}
