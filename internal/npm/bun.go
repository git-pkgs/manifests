package npm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"
)

func init() {
	core.Register("npm", core.Lockfile, &bunLockParser{}, core.ExactMatch("bun.lock"))
}

// bunLockParser parses bun.lock files.
type bunLockParser struct{}

type bunLock struct {
	LockfileVersion int                        `json:"lockfileVersion"`
	Packages        map[string]json.RawMessage `json:"packages"`
	Workspaces      map[string]bunWorkspace    `json:"workspaces"`
}

type bunWorkspace struct {
	Dependencies         map[string]string `json:"dependencies"`
	DevDependencies      map[string]string `json:"devDependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
}

var bunTrailingCommaRegex = regexp.MustCompile(`,(\s*[}\]])`)

func (p *bunLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	// bun.lock uses JSONC (trailing commas allowed), strip them for standard JSON parsing
	cleaned := bunTrailingCommaRegex.ReplaceAll(content, []byte("$1"))

	var lock bunLock
	if err := json.Unmarshal(cleaned, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	// Parse packages section
	// Format: "name": ["name@version", "url", {...}, "integrity"]
	for key, raw := range lock.Packages {
		var arr []any
		if err := json.Unmarshal(raw, &arr); err != nil || len(arr) < 1 {
			continue
		}

		// First element is "name@version"
		nameVersion, ok := arr[0].(string)
		if !ok {
			continue
		}

		name, version := parseBunPackageKey(nameVersion)
		if name == "" {
			// Use the key as name if we can't parse
			name = key
		}

		if seen[name] {
			continue
		}
		seen[name] = true

		// Get integrity if present (last element if it's a string starting with sha)
		var integrity string
		if len(arr) >= 4 {
			if intStr, ok := arr[3].(string); ok && strings.HasPrefix(intStr, "sha") {
				integrity = intStr
			}
		}

		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:   core.Runtime,
			Direct:    false,
			Integrity: integrity,
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
