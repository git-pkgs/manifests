package gem

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"
)

type gemDepKey struct {
	name    string
	version string
}

// stripPlatformSuffix removes Ruby platform suffixes from version strings.
// e.g., "2.4.0-x86_64-linux" -> "2.4.0"
// Based on Gem::Platform from RubyGems source.
func stripPlatformSuffix(version string) string {
	idx := strings.IndexByte(version, '-')
	if idx < 1 {
		return version
	}

	suffix := version[idx+1:]
	if isPlatformSuffix(suffix) {
		return version[:idx]
	}

	return version
}

// isPlatformSuffix checks if a string is a Ruby platform suffix.
// Platform suffixes are either cpu-os (like x86_64-linux) or
// platform-only names (like java, jruby).
func isPlatformSuffix(suffix string) bool {
	// CPU-based platforms start with arch + hyphen
	cpuPrefixes := []string{
		"x86_64-", "x86-", "x64-",
		"i386-", "i486-", "i586-", "i686-",
		"arm64-", "aarch64-", "arm-", "armv",
		"powerpc64-", "powerpc-", "ppc64-", "ppc64le-",
		"sparc-", "s390-", "s390x-",
		"universal-",
	}
	for _, prefix := range cpuPrefixes {
		if strings.HasPrefix(suffix, prefix) {
			return true
		}
	}

	// Platform-only names (no CPU prefix)
	return suffix == "java" || suffix == "jruby" ||
		suffix == "dalvik" || suffix == "dotnet"
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

const gemSpecMinLen = 5 // 4 spaces + at least 1 char

// extractGemSpec extracts gem name and version from "    name (version)" line
func extractGemSpec(line string) (name, version string, ok bool) {
	// Must start with exactly 4 spaces (not 6 - those are sub-deps)
	if len(line) < gemSpecMinLen || line[0] != ' ' || line[1] != ' ' || line[2] != ' ' || line[3] != ' ' || line[4] == ' ' {
		return "", "", false
	}

	// Find the opening paren
	parenStart := strings.IndexByte(line, '(')
	if parenStart < gemSpecMinLen {
		return "", "", false
	}

	// Find closing paren
	parenEnd := strings.IndexByte(line[parenStart:], ')')
	if parenEnd < 0 {
		return "", "", false
	}

	name = strings.TrimSpace(line[4:parenStart])
	version = stripPlatformSuffix(line[parenStart+1 : parenStart+parenEnd])

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
	version = stripPlatformSuffix(trimmed[parenStart+1 : parenStart+parenEnd])

	// Find sha256=
	shaIdx := strings.Index(trimmed, "sha256=")
	if shaIdx < 0 {
		return "", "", "", false
	}

	hash = trimmed[shaIdx+7:]
	return name, version, hash, true
}

const sectionSource = "source"

// detectSection returns the section name if the line is a section header, or empty string otherwise.
func detectSection(trimmed string) (section string, ok bool) {
	switch trimmed {
	case "GEM", "PATH", "GIT":
		return sectionSource, true
	case "PLATFORMS":
		return "platforms", true
	case "DEPENDENCIES":
		return "dependencies", true
	case "CHECKSUMS":
		return "checksums", true
	case "BUNDLED WITH":
		return "bundled", true
	}
	return "", false
}

// collectSpec adds a gem from the specs section if not already seen.
func collectSpec(line string, remote string, seen map[gemDepKey]bool, deps *[]core.Dependency) {
	name, version, ok := extractGemSpec(line)
	if !ok {
		return
	}
	key := gemDepKey{name, version}
	if seen[key] {
		return
	}
	seen[key] = true
	*deps = append(*deps, core.Dependency{
		Name:        name,
		Version:     version,
		Scope:       core.Runtime,
		Direct:      false,
		RegistryURL: remote,
	})
}

// collectDirectDep records a dependency name from the DEPENDENCIES section.
func collectDirectDep(trimmed string, directDeps map[string]bool) {
	name := trimmed
	if idx := strings.IndexByte(name, ' '); idx > 0 {
		name = name[:idx]
	}
	directDeps[strings.TrimSuffix(name, "!")] = true
}

// applyDirectAndChecksums marks direct deps and attaches integrity hashes.
func applyDirectAndChecksums(deps []core.Dependency, directDeps map[string]bool, checksums map[gemDepKey]string) {
	for i := range deps {
		deps[i].Direct = directDeps[deps[i].Name]
		if hash, ok := checksums[gemDepKey{deps[i].Name, deps[i].Version}]; ok {
			deps[i].Integrity = "sha256-" + hash
		}
	}
}

func (p *gemfileLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))
	checksums := make(map[gemDepKey]string)
	directDeps := make(map[string]bool)
	seen := make(map[gemDepKey]bool)

	section := ""
	currentRemote := ""

	core.ForEachLine(text, func(line string) bool {
		trimmed := strings.TrimSpace(line)

		if s, ok := detectSection(trimmed); ok {
			section = s
			if s == sectionSource {
				currentRemote = ""
			}
			return true
		}

		// In source sections, look for remote: line
		if section == sectionSource {
			if strings.HasPrefix(trimmed, "remote:") {
				currentRemote = strings.TrimSpace(trimmed[7:])
				return true
			}
			if trimmed == "specs:" {
				section = "specs"
			}
			return true
		}

		if section == "specs" {
			collectSpec(line, currentRemote, seen, &deps)
		}

		if section == "dependencies" && trimmed != "" {
			collectDirectDep(trimmed, directDeps)
		}

		if section == "checksums" {
			if name, version, hash, ok := extractChecksum(line); ok {
				checksums[gemDepKey{name, version}] = hash
			}
		}
		return true
	})

	applyDirectAndChecksums(deps, directDeps, checksums)

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
