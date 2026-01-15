package rpm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("rpm", core.Manifest, &rpmSpecParser{}, core.SuffixMatch(".spec"))
}

// rpmSpecParser parses RPM *.spec files.
type rpmSpecParser struct{}

var (
	// Match: BuildRequires: pkg or BuildRequires: pkg >= version
	rpmBuildReqRegex = regexp.MustCompile(`(?i)^BuildRequires:\s*(.+)$`)
	// Match: Requires: pkg or Requires: pkg >= version
	rpmRequiresRegex = regexp.MustCompile(`(?i)^Requires(?:\([^)]*\))?:\s*(.+)$`)
)

func (p *rpmSpecParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		var reqStr string
		var scope core.Scope

		if match := rpmBuildReqRegex.FindStringSubmatch(line); match != nil {
			reqStr = match[1]
			scope = core.Build
		} else if match := rpmRequiresRegex.FindStringSubmatch(line); match != nil {
			reqStr = match[1]
			scope = core.Runtime
		} else {
			continue
		}

		// Parse comma-separated deps
		for _, dep := range strings.Split(reqStr, ",") {
			dep = strings.TrimSpace(dep)
			if dep == "" {
				continue
			}

			name, version := parseRPMDep(dep)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// parseRPMDep parses an RPM dependency like "pkg >= 1.0"
func parseRPMDep(dep string) (name, version string) {
	// Split on comparison operators
	for _, op := range []string{">=", "<=", "==", "=", ">", "<"} {
		if idx := strings.Index(dep, op); idx > 0 {
			name = strings.TrimSpace(dep[:idx])
			version = op + " " + strings.TrimSpace(dep[idx+len(op):])
			return name, version
		}
	}

	return strings.TrimSpace(dep), ""
}
