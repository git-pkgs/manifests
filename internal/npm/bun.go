package npm

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("npm", core.Lockfile, &bunLockParser{}, core.ExactMatch("bun.lock"))
}

// bunLockParser parses bun.lock files.
type bunLockParser struct{}

func (p *bunLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[string]bool)

	lines := strings.Split(string(content), "\n")
	inPackages := false

	for _, line := range lines {
		// Detect packages section
		if strings.HasPrefix(line, `  "packages"`) {
			inPackages = true
			continue
		}

		// End of packages section
		if inPackages && strings.HasPrefix(line, "  }") {
			break
		}

		if !inPackages {
			continue
		}

		dep, ok := parseBunPackageLine(line)
		if !ok {
			continue
		}

		if seen[dep.Name] {
			continue
		}
		seen[dep.Name] = true

		deps = append(deps, dep)
	}

	return deps, nil
}

// parseBunPackageLine parses a single package entry line from a bun.lock file.
// Returns the dependency and true if the line was a valid package entry.
func parseBunPackageLine(line string) (core.Dependency, bool) {
	nameVersion, rest, ok := extractBunArrayFirstElement(line)
	if !ok {
		return core.Dependency{}, false
	}

	name, version := parseBunPackageKey(nameVersion)
	if name == "" {
		return core.Dependency{}, false
	}

	return core.Dependency{
		Name:        name,
		Version:     version,
		Scope:       core.Runtime,
		Direct:      false,
		Integrity:   extractBunIntegrity(line),
		RegistryURL: extractBunRegistryURL(rest),
	}, true
}

// extractBunArrayFirstElement extracts the first quoted element from a bun.lock
// package line. It returns the element value, the remainder of the line after
// the first element's closing quote, and whether extraction succeeded.
func extractBunArrayFirstElement(line string) (string, string, bool) {
	if !strings.HasPrefix(line, `    "`) {
		return "", "", false
	}

	colonIdx := strings.Index(line, `": [`)
	if colonIdx < 0 {
		return "", "", false
	}

	arrayStart := colonIdx + len(`": [`)
	if arrayStart >= len(line) || line[arrayStart] != '"' {
		return "", "", false
	}

	endQuote := strings.IndexByte(line[arrayStart+1:], '"')
	if endQuote < 0 {
		return "", "", false
	}

	nameVersion := line[arrayStart+1 : arrayStart+1+endQuote]
	rest := line[arrayStart+1+endQuote+1:]
	return nameVersion, rest, true
}

// extractBunRegistryURL extracts the second array element (URL) from the
// remainder of a bun.lock package line after the first element.
func extractBunRegistryURL(rest string) string {
	commaIdx := strings.Index(rest, `, "`)
	if commaIdx < 0 {
		return ""
	}
	urlStart := commaIdx + len(`, "`)
	if urlEnd := strings.IndexByte(rest[urlStart:], '"'); urlEnd > 0 {
		return rest[urlStart : urlStart+urlEnd]
	}
	return ""
}

// extractBunIntegrity extracts the integrity hash (sha256- or sha512-) from
// a bun.lock package line.
func extractBunIntegrity(line string) string {
	shaIdx := strings.Index(line, `"sha`)
	if shaIdx <= 0 {
		return ""
	}
	endSha := strings.IndexByte(line[shaIdx+1:], '"')
	if endSha <= 0 {
		return ""
	}
	return line[shaIdx+1 : shaIdx+1+endSha]
}

// parseBunPackageKey parses "name@version" from bun.lock
func parseBunPackageKey(key string) (name, version string) {
	// Handle scoped packages: @scope/name@version
	if strings.HasPrefix(key, "@") {
		// Find the second @ which separates name from version
		rest := key[1:]
		if slashIdx := strings.Index(rest, "/"); slashIdx > 0 {
			afterSlash := rest[slashIdx+1:]
			if atIdx := strings.Index(afterSlash, "@"); atIdx > 0 {
				name = "@" + rest[:slashIdx+1+atIdx]
				version = afterSlash[atIdx+1:]
				return name, version
			}
		}
	} else {
		// Non-scoped: name@version
		if atIdx := strings.LastIndex(key, "@"); atIdx > 0 {
			name = key[:atIdx]
			version = key[atIdx+1:]
			return name, version
		}
	}

	return key, ""
}
