package npm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("npm", core.Lockfile, &yarnLockParser{}, core.ExactMatch("yarn.lock"))
}

// yarnLockParser parses yarn.lock files.
type yarnLockParser struct{}

var (
	// Match package header: "package@version":, package@version:, "alias@npm:pkg@version":
	yarnHeaderRegex = regexp.MustCompile(`^"?(@?[^@"]+)@[^"]*"?:?\s*$`)
	// Match version line
	yarnVersionRegex = regexp.MustCompile(`^\s+version\s+"?([^"\s]+)"?\s*$`)
	// Match resolved line for integrity
	yarnResolvedRegex = regexp.MustCompile(`^\s+resolved\s+"([^"]+)"`)
	// Match integrity line
	yarnIntegrityRegex = regexp.MustCompile(`^\s+integrity\s+(\S+)`)
)

func (p *yarnLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)

	var currentName string
	var currentVersion string
	var currentIntegrity string

	for _, line := range lines {
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Check for package header
		if match := yarnHeaderRegex.FindStringSubmatch(line); match != nil {
			// Save previous package if we have one
			if currentName != "" && currentVersion != "" && !seen[currentName] {
				seen[currentName] = true
				deps = append(deps, core.Dependency{
					Name:      currentName,
					Version:   currentVersion,
					Scope:   core.Runtime,
					Direct:    false,
					Integrity: currentIntegrity,
				})
			}

			// Start new package - extract actual package name from header
			currentName = extractYarnPackageName(match[1])
			currentVersion = ""
			currentIntegrity = ""
			continue
		}

		// Check for version
		if match := yarnVersionRegex.FindStringSubmatch(line); match != nil {
			currentVersion = match[1]
			continue
		}

		// Check for integrity
		if match := yarnIntegrityRegex.FindStringSubmatch(line); match != nil {
			currentIntegrity = match[1]
		}
	}

	// Don't forget the last package
	if currentName != "" && currentVersion != "" && !seen[currentName] {
		deps = append(deps, core.Dependency{
			Name:      currentName,
			Version:   currentVersion,
			Scope:   core.Runtime,
			Direct:    false,
			Integrity: currentIntegrity,
		})
	}

	return deps, nil
}

// extractYarnPackageName extracts the package name from yarn header patterns
func extractYarnPackageName(header string) string {
	// Handle npm: alias pattern like "alias@npm:@scope/pkg"
	if idx := strings.Index(header, "@npm:"); idx > 0 {
		return strings.TrimPrefix(header[idx:], "@npm:")
	}
	return header
}
