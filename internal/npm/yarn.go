package npm

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("npm", core.Lockfile, &yarnLockParser{}, core.ExactMatch("yarn.lock"))
}

// yarnLockParser parses yarn.lock files (both v1 and v4 formats).
type yarnLockParser struct{}

func (p *yarnLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)

	// Check if this is a v4 lockfile
	isV4 := strings.Contains(string(content), "__metadata:")

	var currentName string
	var currentVersion string
	var currentIntegrity string
	var currentResolved string

	for _, line := range lines {
		// Skip empty lines
		if len(line) == 0 {
			continue
		}

		// Skip comments
		if line[0] == '#' {
			continue
		}

		// Skip __metadata and workspace entries
		if strings.HasPrefix(line, "__metadata:") {
			continue
		}

		// Package header: no leading whitespace, ends with :
		if line[0] != ' ' && line[0] != '\t' {
			// Save previous package if we have one
			if currentName != "" && currentVersion != "" && !seen[currentName] {
				seen[currentName] = true
				deps = append(deps, core.Dependency{
					Name:        currentName,
					Version:     currentVersion,
					Scope:       core.Runtime,
					Direct:      false,
					Integrity:   currentIntegrity,
					RegistryURL: currentResolved,
				})
			}

			// Parse header: "package@version": or package@version:
			currentName = parseYarnHeader(line)
			currentVersion = ""
			currentIntegrity = ""
			currentResolved = ""
			continue
		}

		// Indented lines are package details
		trimmed := strings.TrimLeft(line, " \t")

		// Check for version line
		if strings.HasPrefix(trimmed, "version") {
			currentVersion = extractYarnValue(trimmed[7:])
			continue
		}

		// Check for resolved line (v1)
		if strings.HasPrefix(trimmed, "resolved ") {
			currentResolved = extractYarnValue(trimmed[9:])
			continue
		}

		// Check for resolution line (v4)
		if isV4 && strings.HasPrefix(trimmed, "resolution:") {
			res := extractYarnValue(trimmed[11:])
			// v4 resolution can be "pkg@npm:version" or a URL
			if strings.HasPrefix(res, "http") {
				currentResolved = res
			}
			continue
		}

		// Check for integrity (v1)
		if strings.HasPrefix(trimmed, "integrity ") {
			currentIntegrity = strings.TrimSpace(trimmed[10:])
			continue
		}

		// Check for checksum (v4)
		if isV4 && strings.HasPrefix(trimmed, "checksum") {
			checksum := extractYarnValue(trimmed[8:])
			// v4 checksums are in format: 10c0/hash... or sha512-...
			if idx := strings.IndexByte(checksum, '/'); idx > 0 {
				currentIntegrity = "sha512-" + checksum[idx+1:]
			} else if strings.HasPrefix(checksum, "sha") {
				currentIntegrity = checksum
			}
			continue
		}
	}

	// Don't forget the last package
	if currentName != "" && currentVersion != "" && !seen[currentName] {
		// Skip workspace packages
		if !strings.HasPrefix(currentVersion, "0.0.0-use.local") {
			deps = append(deps, core.Dependency{
				Name:        currentName,
				Version:     currentVersion,
				Scope:       core.Runtime,
				Direct:      false,
				Integrity:   currentIntegrity,
				RegistryURL: currentResolved,
			})
		}
	}

	return deps, nil
}

// parseYarnHeader extracts the package name from a yarn header line.
// Formats: "package@version":, package@version:, "alias@npm:@scope/pkg@version":
func parseYarnHeader(line string) string {
	// Remove leading quote if present
	if len(line) > 0 && line[0] == '"' {
		line = line[1:]
	}

	// Find the @ that separates name from version
	// For scoped packages like @scope/pkg@version, we need to find the second @
	atIdx := strings.IndexByte(line, '@')
	if atIdx < 0 {
		return ""
	}

	// If scoped package, find the next @
	if atIdx == 0 {
		nextAt := strings.IndexByte(line[1:], '@')
		if nextAt < 0 {
			return ""
		}
		atIdx = nextAt + 1
	}

	name := line[:atIdx]

	// Handle npm: alias pattern like "alias@npm:@scope/pkg"
	if idx := strings.Index(name, "@npm:"); idx > 0 {
		return name[idx+5:]
	}

	return name
}

// extractYarnValue extracts a value after a key, handling both quoted and unquoted formats.
// Input: `: "value"` or ` "value"` or ` value`
func extractYarnValue(s string) string {
	s = strings.TrimLeft(s, " \t:")
	if len(s) == 0 {
		return ""
	}
	if s[0] == '"' {
		// Find closing quote
		end := strings.IndexByte(s[1:], '"')
		if end < 0 {
			return s[1:]
		}
		return s[1 : end+1]
	}
	// Unquoted: take until whitespace
	if end := strings.IndexAny(s, " \t"); end > 0 {
		return s[:end]
	}
	return s
}

// extractYarnPackageName extracts the package name from yarn header patterns
func extractYarnPackageName(header string) string {
	// Handle npm: alias pattern like "alias@npm:@scope/pkg"
	if idx := strings.Index(header, "@npm:"); idx > 0 {
		return strings.TrimPrefix(header[idx:], "@npm:")
	}
	return header
}
