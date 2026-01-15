package gem

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"
)

type gemDepKey struct {
	name    string
	version string
}

func init() {
	// Gemfile and gems.rb - manifests
	core.Register("gem", core.Manifest, &gemfileParser{}, core.ExactMatch("Gemfile", "gems.rb"))

	// Gemfile.lock and gems.locked - lockfiles
	core.Register("gem", core.Lockfile, &gemfileLockParser{}, core.SuffixMatch("Gemfile.lock", "gems.locked"))

	// gemspec files - manifests
	core.Register("gem", core.Manifest, &gemspecParser{}, core.SuffixMatch(".gemspec"))
}

// gemfileParser parses Gemfile and gems.rb files.
type gemfileParser struct{}

// extractGemDecl extracts gem name and version from a gem declaration line
// Handles: gem "name" or gem "name", "version"
func extractGemDecl(line string) (name, version string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "gem ") && !strings.HasPrefix(trimmed, "gem\t") {
		return "", "", false
	}

	// Find first quote
	start := strings.IndexAny(trimmed, "'\"")
	if start < 0 {
		return "", "", false
	}
	quote := trimmed[start]

	// Find end of name
	end := strings.IndexByte(trimmed[start+1:], quote)
	if end < 0 {
		return "", "", false
	}
	name = trimmed[start+1 : start+1+end]

	// Look for version
	rest := trimmed[start+1+end+1:]
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		rest = rest[idx+1:]
		// Find version quote
		vstart := strings.IndexAny(rest, "'\"")
		if vstart >= 0 {
			vquote := rest[vstart]
			vend := strings.IndexByte(rest[vstart+1:], vquote)
			if vend >= 0 {
				version = rest[vstart+1 : vstart+1+vend]
			}
		}
	}

	return name, version, true
}

// extractGemfileGroup extracts scope from group declaration
func extractGemfileGroup(line string) (scope core.Scope, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "group ") {
		return core.Runtime, false
	}
	if !strings.HasSuffix(trimmed, " do") {
		return core.Runtime, false
	}

	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, ":development") || strings.Contains(lower, ":dev") {
		return core.Development, true
	}
	if strings.Contains(lower, ":test") {
		return core.Test, true
	}
	return core.Runtime, true
}

func (p *gemfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	currentScope := core.Runtime
	groupDepth := 0

	core.ForEachLine(text, func(line string) bool {
		trimmed := strings.TrimSpace(line)

		// Track group blocks
		if scope, ok := extractGemfileGroup(line); ok {
			groupDepth++
			currentScope = scope
			return true
		}

		if trimmed == "end" {
			groupDepth--
			if groupDepth == 0 {
				currentScope = core.Runtime
			}
			return true
		}

		// Parse gem declarations
		if name, version, ok := extractGemDecl(line); ok {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   currentScope,
				Direct:  true,
			})
		}
		return true
	})

	return deps, nil
}

// gemfileLockParser parses Gemfile.lock files.
type gemfileLockParser struct{}

// extractGemSpec extracts gem name and version from "    name (version)" line
func extractGemSpec(line string) (name, version string, ok bool) {
	// Must start with exactly 4 spaces (not 6 - those are sub-deps)
	if len(line) < 5 || line[0] != ' ' || line[1] != ' ' || line[2] != ' ' || line[3] != ' ' || line[4] == ' ' {
		return "", "", false
	}

	// Find the opening paren
	parenStart := strings.IndexByte(line, '(')
	if parenStart < 5 {
		return "", "", false
	}

	// Find closing paren
	parenEnd := strings.IndexByte(line[parenStart:], ')')
	if parenEnd < 0 {
		return "", "", false
	}

	name = strings.TrimSpace(line[4:parenStart])
	version = line[parenStart+1 : parenStart+parenEnd]

	return name, version, true
}

// extractChecksum extracts name, version, sha256 from checksum line
func extractChecksum(line string) (name, version, hash string, ok bool) {
	trimmed := strings.TrimSpace(line)

	// Find paren for version
	parenStart := strings.IndexByte(trimmed, '(')
	if parenStart < 1 {
		return "", "", "", false
	}
	parenEnd := strings.IndexByte(trimmed[parenStart:], ')')
	if parenEnd < 0 {
		return "", "", "", false
	}

	name = trimmed[:parenStart-1]
	version = trimmed[parenStart+1 : parenStart+parenEnd]

	// Find sha256=
	shaIdx := strings.Index(trimmed, "sha256=")
	if shaIdx < 0 {
		return "", "", "", false
	}

	hash = trimmed[shaIdx+7:]
	return name, version, hash, true
}

func (p *gemfileLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))
	checksums := make(map[gemDepKey]string)
	directDeps := make(map[string]bool)

	section := ""

	core.ForEachLine(text, func(line string) bool {
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		switch trimmed {
		case "GEM", "PATH", "GIT":
			section = "specs"
			return true
		case "PLATFORMS":
			section = "platforms"
			return true
		case "DEPENDENCIES":
			section = "dependencies"
			return true
		case "CHECKSUMS":
			section = "checksums"
			return true
		case "BUNDLED WITH":
			section = "bundled"
			return true
		}

		if section == "specs" {
			if name, version, ok := extractGemSpec(line); ok {
				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
					Scope:   core.Runtime,
					Direct:  false,
				})
			}
		}

		if section == "dependencies" && trimmed != "" {
			name := trimmed
			if idx := strings.IndexByte(name, ' '); idx > 0 {
				name = name[:idx]
			}
			name = strings.TrimSuffix(name, "!")
			directDeps[name] = true
		}

		if section == "checksums" {
			if name, version, hash, ok := extractChecksum(line); ok {
				checksums[gemDepKey{name, version}] = hash
			}
		}
		return true
	})

	// Update direct status and checksums
	for i := range deps {
		deps[i].Direct = directDeps[deps[i].Name]
		if hash, ok := checksums[gemDepKey{deps[i].Name, deps[i].Version}]; ok {
			deps[i].Integrity = "sha256-" + hash
		}
	}

	return deps, nil
}

// gemspecParser parses .gemspec files.
type gemspecParser struct{}

// extractGemspecDep extracts dependency from add_dependency or add_development_dependency line
func extractGemspecDep(line string) (name, version string, isDev bool, ok bool) {
	// Look for .add_dependency, .add_runtime_dependency, or .add_development_dependency
	var idx int
	if idx = strings.Index(line, ".add_development_dependency"); idx >= 0 {
		isDev = true
		idx += len(".add_development_dependency")
	} else if idx = strings.Index(line, ".add_runtime_dependency"); idx >= 0 {
		idx += len(".add_runtime_dependency")
	} else if idx = strings.Index(line, ".add_dependency"); idx >= 0 {
		idx += len(".add_dependency")
	} else {
		return "", "", false, false
	}

	rest := line[idx:]

	// Find opening paren or quote
	start := strings.IndexAny(rest, "('\"")
	if start < 0 {
		return "", "", false, false
	}

	// Skip paren if present
	if rest[start] == '(' {
		rest = rest[start+1:]
		start = strings.IndexAny(rest, "'\"")
		if start < 0 {
			return "", "", false, false
		}
	}

	quote := rest[start]
	end := strings.IndexByte(rest[start+1:], quote)
	if end < 0 {
		return "", "", false, false
	}
	name = rest[start+1 : start+1+end]

	// Look for version
	rest = rest[start+1+end+1:]
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		rest = rest[idx+1:]
		vstart := strings.IndexAny(rest, "'\"")
		if vstart >= 0 {
			vquote := rest[vstart]
			vend := strings.IndexByte(rest[vstart+1:], vquote)
			if vend >= 0 {
				version = rest[vstart+1 : vstart+1+vend]
			}
		}
	}

	return name, version, isDev, true
}

func (p *gemspecParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	core.ForEachLine(text, func(line string) bool {
		if name, version, isDev, ok := extractGemspecDep(line); ok {
			scope := core.Runtime
			if isDev {
				scope = core.Development
			}
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
		}
		return true
	})

	return deps, nil
}
