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

		// Package line format: "name": ["resolved@version", "url", {...}, "sha..."],
		// Look for lines starting with 4 spaces + quote
		if !strings.HasPrefix(line, `    "`) {
			continue
		}

		// Extract the array part after the colon
		colonIdx := strings.Index(line, `": [`)
		if colonIdx < 0 {
			continue
		}

		// Find first element in array (name@version)
		arrayStart := colonIdx + 4
		if arrayStart >= len(line) {
			continue
		}

		// First element is "name@version"
		if line[arrayStart] != '"' {
			continue
		}

		// Find closing quote of first element
		endQuote := strings.IndexByte(line[arrayStart+1:], '"')
		if endQuote < 0 {
			continue
		}

		nameVersion := line[arrayStart+1 : arrayStart+1+endQuote]
		name, version := parseBunPackageKey(nameVersion)
		if name == "" {
			continue
		}

		if seen[name] {
			continue
		}
		seen[name] = true

		// Extract second element (URL) - it follows the first element after ", "
		var registryURL string
		firstElemEnd := arrayStart + 1 + endQuote + 1 // position after closing quote
		rest := line[firstElemEnd:]
		// Look for next quoted string after comma
		if commaIdx := strings.Index(rest, `, "`); commaIdx >= 0 {
			urlStart := commaIdx + 3 // skip `, "`
			if urlEnd := strings.IndexByte(rest[urlStart:], '"'); urlEnd > 0 {
				registryURL = rest[urlStart : urlStart+urlEnd]
			}
		}

		// Look for integrity hash (sha256- or sha512-)
		var integrity string
		if shaIdx := strings.Index(line, `"sha`); shaIdx > 0 {
			// Find closing quote
			endSha := strings.IndexByte(line[shaIdx+1:], '"')
			if endSha > 0 {
				integrity = line[shaIdx+1 : shaIdx+1+endSha]
			}
		}

		deps = append(deps, core.Dependency{
			Name:        name,
			Version:     version,
			Scope:       core.Runtime,
			Direct:      false,
			Integrity:   integrity,
			RegistryURL: registryURL,
		})
	}

	return deps, nil
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
