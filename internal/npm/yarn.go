package npm

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("npm", core.Lockfile, &yarnLockParser{}, core.ExactMatch("yarn.lock"))
}

// yarnLockParser parses yarn.lock files (both v1 and v4 formats).
type yarnLockParser struct{}

var (
	// Match package header: "package@version":, package@version:, "alias@npm:pkg@version":
	yarnHeaderRegex = regexp.MustCompile(`^"?(@?[^@"]+)@[^"]*"?:?\s*$`)
	// Match version line
	yarnVersionRegex = regexp.MustCompile(`^\s+version:?\s+"?([^"\s]+)"?\s*$`)
	// Match resolved line for integrity
	yarnResolvedRegex = regexp.MustCompile(`^\s+resolved\s+"([^"]+)"`)
	// Match integrity line (v1)
	yarnIntegrityRegex = regexp.MustCompile(`^\s+integrity\s+(\S+)`)
	// Match checksum line (v4)
	yarnChecksumRegex = regexp.MustCompile(`^\s+checksum:?\s+(\S+)`)
	// Match resolution line (v4)
	yarnResolutionRegex = regexp.MustCompile(`^\s+resolution:?\s+"([^"]+)"`)
)

func (p *yarnLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)

	// Check if this is a v4 lockfile
	isV4 := strings.Contains(string(content), "__metadata:")

	var currentName string
	var currentVersion string
	var currentIntegrity string

	for _, line := range lines {
		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Skip __metadata and workspace entries
		if strings.HasPrefix(line, "__metadata:") {
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
					Scope:     core.Runtime,
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

		// Check for integrity (v1)
		if match := yarnIntegrityRegex.FindStringSubmatch(line); match != nil {
			currentIntegrity = match[1]
			continue
		}

		// Check for checksum (v4) - convert to sha512 format
		if isV4 {
			if match := yarnChecksumRegex.FindStringSubmatch(line); match != nil {
				checksum := match[1]
				// v4 checksums are in format: 10c0/hash... or sha512-...
				if strings.Contains(checksum, "/") {
					parts := strings.SplitN(checksum, "/", 2)
					if len(parts) == 2 {
						currentIntegrity = "sha512-" + parts[1]
					}
				} else if strings.HasPrefix(checksum, "sha") {
					currentIntegrity = checksum
				}
				continue
			}
		}
	}

	// Don't forget the last package
	if currentName != "" && currentVersion != "" && !seen[currentName] {
		// Skip workspace packages
		if !strings.HasPrefix(currentVersion, "0.0.0-use.local") {
			deps = append(deps, core.Dependency{
				Name:      currentName,
				Version:   currentVersion,
				Scope:     core.Runtime,
				Direct:    false,
				Integrity: currentIntegrity,
			})
		}
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
