package npm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"
)

func init() {
	core.Register("npm", core.Lockfile, &pnpmLockParser{}, core.ExactMatch("pnpm-lock.yaml"))
}

// extractPnpmPackageKey extracts package key from "  /name/ver:" or "  '@scope/name@ver':" lines
func extractPnpmPackageKey(line string) (string, bool) {
	// Must start with 2 spaces
	if len(line) < 4 || line[0] != ' ' || line[1] != ' ' {
		return "", false
	}
	// Must end with colon
	if line[len(line)-1] != ':' {
		return "", false
	}
	key := line[2 : len(line)-1]
	// Remove surrounding quotes if present
	if len(key) >= 2 && (key[0] == '\'' || key[0] == '"') {
		key = key[1 : len(key)-1]
	}
	// Must start with / or @
	if len(key) == 0 || (key[0] != '/' && key[0] != '@') {
		return "", false
	}
	return key, true
}

// extractPnpmIntegrity extracts sha hash from "integrity: shaXXX" pattern
func extractPnpmIntegrity(line string) (string, bool) {
	idx := strings.Index(line, "integrity: sha")
	if idx < 0 {
		return "", false
	}
	start := idx + len("integrity: ")
	// Find end of hash (space or end of line or closing brace)
	rest := line[start:]
	end := strings.IndexAny(rest, " ,}")
	if end < 0 {
		return rest, true
	}
	return rest[:end], true
}

// extractPnpmTarball extracts tarball URL from "tarball: <url>" pattern
func extractPnpmTarball(line string) (string, bool) {
	// Check for "tarball:" pattern (v6+ format)
	if idx := strings.Index(line, "tarball:"); idx >= 0 {
		rest := strings.TrimSpace(line[idx+8:])
		// Remove surrounding quotes if present
		if len(rest) >= 2 && (rest[0] == '\'' || rest[0] == '"') {
			rest = rest[1 : len(rest)-1]
		}
		if strings.HasPrefix(rest, "http") {
			return rest, true
		}
	}
	return "", false
}

// pnpmLockParser parses pnpm-lock.yaml files using regex for speed.
type pnpmLockParser struct{}

func (p *pnpmLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))
	seen := make(map[string]bool)

	inPackages := false
	var currentKey string
	var currentIntegrity string
	var currentTarball string
	var currentDev bool

	core.ForEachLine(text, func(line string) bool {
		// Detect packages: or snapshots: section
		if line == "packages:" || line == "snapshots:" {
			inPackages = true
			return true
		}

		// End of packages section (new top-level key)
		if inPackages && len(line) > 0 && line[0] != ' ' && line[0] != '\n' {
			inPackages = false
			currentKey = ""
		}

		if !inPackages {
			return true
		}

		// Package key line (2-space indent)
		if key, ok := extractPnpmPackageKey(line); ok {
			// Save previous package if any
			if currentKey != "" {
				name, version := parsePnpmPackageKey(currentKey)
				if name != "" && !seen[name] {
					seen[name] = true
					scope := core.Runtime
					if currentDev {
						scope = core.Development
					}
					deps = append(deps, core.Dependency{
						Name:        name,
						Version:     version,
						Scope:       scope,
						Direct:      false,
						Integrity:   currentIntegrity,
						RegistryURL: currentTarball,
					})
				}
			}
			currentKey = key
			currentIntegrity = ""
			currentTarball = ""
			currentDev = false
			return true
		}

		// Look for integrity, tarball, and dev within package block
		if currentKey != "" {
			if integrity, ok := extractPnpmIntegrity(line); ok {
				currentIntegrity = integrity
			}
			if tarball, ok := extractPnpmTarball(line); ok {
				currentTarball = tarball
			}
			if strings.Contains(line, "dev: true") {
				currentDev = true
			}
		}
		return true
	})

	// Don't forget the last package
	if currentKey != "" {
		name, version := parsePnpmPackageKey(currentKey)
		if name != "" && !seen[name] {
			seen[name] = true
			scope := core.Runtime
			if currentDev {
				scope = core.Development
			}
			deps = append(deps, core.Dependency{
				Name:        name,
				Version:     version,
				Scope:       scope,
				Direct:      false,
				Integrity:   currentIntegrity,
				RegistryURL: currentTarball,
			})
		}
	}

	return deps, nil
}

// parsePnpmPackageKey parses a pnpm package key like "/@scope/name/1.0.0" or "@scope/name@1.0.0"
func parsePnpmPackageKey(key string) (name, version string) {
	key = strings.TrimPrefix(key, "/")

	// Format varies by lockfile version:
	// v5: /@scope/name/1.0.0 or /name/1.0.0
	// v6+: @scope/name@1.0.0 or name@1.0.0

	// Handle scoped packages
	if strings.HasPrefix(key, "@") {
		// Find the second @ (version separator) or second / (v5 style)
		rest := key[1:] // skip first @
		if slashIdx := strings.Index(rest, "/"); slashIdx > 0 {
			afterScope := rest[slashIdx+1:]
			// Check if this is v5 style (@scope/name/version)
			if nextSlash := strings.Index(afterScope, "/"); nextSlash > 0 {
				name = "@" + rest[:slashIdx+1+nextSlash]
				version = afterScope[nextSlash+1:]
				// Remove any suffixes like (react@18.2.0)
				if parenIdx := strings.Index(version, "("); parenIdx > 0 {
					version = version[:parenIdx]
				}
				return name, version
			}
			// Check for v6+ style (@scope/name@version)
			if atIdx := strings.Index(afterScope, "@"); atIdx > 0 {
				name = "@" + rest[:slashIdx+1+atIdx]
				version = afterScope[atIdx+1:]
				if parenIdx := strings.Index(version, "("); parenIdx > 0 {
					version = version[:parenIdx]
				}
				return name, version
			}
		}
	} else {
		// Non-scoped package
		// v5 style: name/version
		if slashIdx := strings.Index(key, "/"); slashIdx > 0 {
			name = key[:slashIdx]
			version = key[slashIdx+1:]
			if parenIdx := strings.Index(version, "("); parenIdx > 0 {
				version = version[:parenIdx]
			}
			return name, version
		}
		// v6+ style: name@version
		if atIdx := strings.LastIndex(key, "@"); atIdx > 0 {
			name = key[:atIdx]
			version = key[atIdx+1:]
			if parenIdx := strings.Index(version, "("); parenIdx > 0 {
				version = version[:parenIdx]
			}
			return name, version
		}
	}

	return "", ""
}
