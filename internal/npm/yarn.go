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

// yarnParseState tracks the current package being parsed.
type yarnParseState struct {
	name      string
	version   string
	integrity string
	resolved  string
}

func (s *yarnParseState) reset(name string) {
	s.name = name
	s.version = ""
	s.integrity = ""
	s.resolved = ""
}

// collectDep appends the current state as a dependency if it has a name and version
// that haven't been seen yet. Returns the updated deps slice and seen map.
func (s *yarnParseState) collectDep(deps []core.Dependency, seen map[string]bool) []core.Dependency {
	if s.name == "" || s.version == "" || seen[s.name] {
		return deps
	}
	seen[s.name] = true
	return append(deps, core.Dependency{
		Name:        s.name,
		Version:     s.version,
		Scope:       core.Runtime,
		Direct:      false,
		Integrity:   s.integrity,
		RegistryURL: s.resolved,
	})
}

func (p *yarnLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)
	isV4 := strings.Contains(string(content), "__metadata:")

	var state yarnParseState

	for _, line := range lines {
		if skipYarnLine(line) {
			continue
		}

		if isYarnHeader(line) {
			deps = state.collectDep(deps, seen)
			state.reset(parseYarnHeader(line))
			continue
		}

		parseYarnDetailLine(strings.TrimLeft(line, " \t"), isV4, &state)
	}

	// Don't forget the last package
	if state.name != "" && state.version != "" && !seen[state.name] {
		if !strings.HasPrefix(state.version, "0.0.0-use.local") {
			deps = state.collectDep(deps, seen)
		}
	}

	return deps, nil
}

// skipYarnLine returns true for lines that should be ignored: empty lines,
// comments, and __metadata entries.
func skipYarnLine(line string) bool {
	if len(line) == 0 {
		return true
	}
	if line[0] == '#' {
		return true
	}
	return strings.HasPrefix(line, "__metadata:")
}

// isYarnHeader returns true if the line is a package header (no leading whitespace).
func isYarnHeader(line string) bool {
	return line[0] != ' ' && line[0] != '\t'
}

// parseYarnDetailLine extracts version, resolved, integrity, or checksum
// from an indented detail line and updates state accordingly.
func parseYarnDetailLine(trimmed string, isV4 bool, state *yarnParseState) {
	if strings.HasPrefix(trimmed, "version") {
		state.version = extractYarnValue(trimmed[7:])
		return
	}

	if strings.HasPrefix(trimmed, "resolved ") {
		state.resolved = extractYarnValue(trimmed[9:])
		return
	}

	if isV4 && strings.HasPrefix(trimmed, "resolution:") {
		res := extractYarnValue(trimmed[11:])
		if strings.HasPrefix(res, "http") {
			state.resolved = res
		}
		return
	}

	if strings.HasPrefix(trimmed, "integrity ") {
		state.integrity = strings.TrimSpace(trimmed[10:])
		return
	}

	if isV4 && strings.HasPrefix(trimmed, "checksum") {
		state.integrity = parseV4Checksum(extractYarnValue(trimmed[8:]))
	}
}

// parseV4Checksum converts a v4 checksum value into an integrity string.
// v4 checksums use the format "10c0/hash..." or "sha512-...".
func parseV4Checksum(checksum string) string {
	if idx := strings.IndexByte(checksum, '/'); idx > 0 {
		return "sha512-" + checksum[idx+1:]
	}
	if strings.HasPrefix(checksum, "sha") {
		return checksum
	}
	return ""
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
