package maven

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"
)

func init() {
	core.Register("maven", core.Manifest, &gradleParser{}, core.ExactMatch("build.gradle"))
	core.Register("maven", core.Manifest, &gradleParser{}, core.ExactMatch("build.gradle.kts"))
	core.Register("maven", core.Lockfile, &gradleLockfileParser{}, core.ExactMatch("gradle.lockfile"))
}

// gradleParser parses build.gradle and build.gradle.kts files.
type gradleParser struct{}

// gradleKeywords are dependency declaration keywords
var gradleKeywords = []string{
	"compile", "implementation", "api", "runtimeOnly", "compileOnly",
	"testCompile", "testImplementation", "testRuntimeOnly",
	"annotationProcessor", "kapt",
}

// findGradleKeyword checks if line contains a gradle dependency keyword
// Returns the keyword and its position, or empty string if not found
func findGradleKeyword(line string) (keyword string, pos int, isTest bool) {
	lower := strings.ToLower(line)
	for _, kw := range gradleKeywords {
		if idx := strings.Index(lower, kw); idx >= 0 {
			// Check it's at word boundary (space or start)
			if idx > 0 && line[idx-1] != ' ' && line[idx-1] != '\t' {
				continue
			}
			return kw, idx, strings.Contains(kw, "test")
		}
	}
	return "", -1, false
}

// extractGradleCoords extracts group:artifact:version from a gradle dependency line
func extractGradleCoords(line string, keywordEnd int) (group, artifact, version string, ok bool) {
	rest := line[keywordEnd:]

	// Find opening quote or paren
	start := strings.IndexAny(rest, "'\"(")
	if start < 0 {
		return "", "", "", false
	}

	opener := rest[start]
	closer := byte('\'')
	if opener == '"' {
		closer = '"'
	} else if opener == '(' {
		closer = ')'
	}

	// Find closing quote
	end := strings.IndexByte(rest[start+1:], closer)
	if end < 0 {
		return "", "", "", false
	}

	coords := rest[start+1 : start+1+end]

	// Skip variable references and non-maven coords
	if strings.Contains(coords, "$") || strings.Contains(coords, "project(") || strings.Contains(coords, "files(") {
		return "", "", "", false
	}

	// Parse group:artifact:version
	parts := strings.Split(coords, ":")
	if len(parts) < 3 {
		return "", "", "", false
	}

	return parts[0], parts[1], parts[2], true
}

// extractGradleMapForm extracts deps from: compile group: 'x', name: 'y', version: 'z'
func extractGradleMapForm(line string) (group, artifact, version string, ok bool) {
	// Check for group:, name:, version: pattern
	groupIdx := strings.Index(line, "group:")
	nameIdx := strings.Index(line, "name:")
	versionIdx := strings.Index(line, "version:")

	if groupIdx < 0 || nameIdx < 0 || versionIdx < 0 {
		return "", "", "", false
	}

	group = extractQuotedAfter(line[groupIdx+6:])
	artifact = extractQuotedAfter(line[nameIdx+5:])
	version = extractQuotedAfter(line[versionIdx+8:])

	if group == "" || artifact == "" || version == "" {
		return "", "", "", false
	}

	return group, artifact, version, true
}

// extractQuotedAfter extracts the first quoted string from s
func extractQuotedAfter(s string) string {
	start := strings.IndexAny(s, "'\"")
	if start < 0 {
		return ""
	}
	quote := s[start]
	end := strings.IndexByte(s[start+1:], quote)
	if end < 0 {
		return ""
	}
	return s[start+1 : start+1+end]
}

func (p *gradleParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))
	seen := make(map[string]bool)

	core.ForEachLine(text, func(line string) bool {
		keyword, pos, isTest := findGradleKeyword(line)
		if keyword == "" {
			return true
		}

		// Try short form first: compile 'group:artifact:version'
		group, artifact, version, ok := extractGradleCoords(line, pos+len(keyword))
		if !ok {
			// Try map form: compile group: 'x', name: 'y', version: 'z'
			group, artifact, version, ok = extractGradleMapForm(line)
			if !ok {
				return true
			}
		}

		name := group + ":" + artifact
		if seen[name] {
			return true
		}
		seen[name] = true

		scope := core.Runtime
		if isTest {
			scope = core.Test
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   scope,
			Direct:  true,
		})
		return true
	})

	return deps, nil
}

// gradleLockfileParser parses gradle.lockfile files.
type gradleLockfileParser struct{}

func (p *gradleLockfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	core.ForEachLine(text, func(line string) bool {
		// Skip leading whitespace
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || line[0] == '#' || line == "empty=" {
			return true
		}

		// Format: group:artifact:version=configurations
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			return true
		}

		coords := line[:idx]
		// Find first two colons for group:artifact:version
		firstColon := strings.IndexByte(coords, ':')
		if firstColon < 0 {
			return true
		}
		secondColon := strings.IndexByte(coords[firstColon+1:], ':')
		if secondColon < 0 {
			return true
		}
		secondColon += firstColon + 1

		name := coords[:secondColon]
		version := coords[secondColon+1:]

		scope := core.Runtime
		configs := line[idx+1:]
		if strings.Contains(configs, "test") {
			scope = core.Test
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   scope,
			Direct:  false,
		})
		return true
	})

	return deps, nil
}
