package maven

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("maven", core.Manifest, &gradleParser{}, core.ExactMatch("build.gradle"))
	core.Register("maven", core.Manifest, &gradleParser{}, core.ExactMatch("build.gradle.kts"))
	core.Register("maven", core.Lockfile, &gradleLockfileParser{}, core.ExactMatch("gradle.lockfile"))

	// gradle-dependencies-q.txt - lockfile (gradle dependencies -q output)
	core.Register("maven", core.Lockfile, &gradleDependenciesParser{}, core.ExactMatch("gradle-dependencies-q.txt"))

	// verification-metadata.xml - lockfile (gradle dependency verification)
	core.Register("maven", core.Lockfile, &gradleVerificationParser{}, core.ExactMatch("verification-metadata.xml"))

	// dependencies.lock - lockfile (Nebula dependency-lock plugin)
	core.Register("maven", core.Lockfile, &nebulaLockParser{}, core.ExactMatch("dependencies.lock"))

	// gradle-html-dependency-report.js - lockfile (gradle htmlDependencyReport task)
	core.Register("maven", core.Lockfile, &gradleHtmlReportParser{}, core.ExactMatch("gradle-html-dependency-report.js"))
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
	var closer byte
	switch opener {
	case '"':
		closer = '"'
	case '(':
		closer = ')'
	default:
		closer = '\''
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

// gradleDependenciesParser parses gradle-dependencies-q.txt files (gradle dependencies -q output).
type gradleDependenciesParser struct{}

// Match lines like: +--- org.group:artifact:version or \--- org.group:artifact:version
// Also matches lines with version resolution: org.group:artifact:1.0 -> 2.0
var gradleDepLineRegex = regexp.MustCompile(`[+\\]---\s+([a-zA-Z0-9._-]+:[a-zA-Z0-9._-]+):([^\s]+)`)

func (p *gradleDependenciesParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[string]bool)
	lines := strings.Split(string(content), "\n")

	inTestConfig := false

	for _, line := range lines {
		// Detect configuration headers
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "test") && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "\\") && !strings.HasPrefix(line, "|") {
			inTestConfig = true
		} else if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "\\") && !strings.HasPrefix(line, "|") && len(strings.TrimSpace(line)) > 0 {
			inTestConfig = false
		}

		if match := gradleDepLineRegex.FindStringSubmatch(line); match != nil {
			name := match[1]
			version := match[2]

			// Handle version resolution: 1.0 -> 2.0
			if idx := strings.LastIndex(version, " -> "); idx >= 0 {
				version = strings.TrimSpace(version[idx+4:])
			}

			// Skip (*) which indicates already shown
			if strings.HasSuffix(version, "(*)") {
				continue
			}
			// Clean up version
			version = strings.TrimSuffix(version, " (*)")
			version = strings.TrimSuffix(version, " (n)")

			if seen[name] {
				continue
			}
			seen[name] = true

			scope := core.Runtime
			if inTestConfig {
				scope = core.Test
			}

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  false,
			})
		}
	}

	return deps, nil
}

// gradleVerificationParser parses verification-metadata.xml files.
type gradleVerificationParser struct{}

type verificationMetadata struct {
	Components struct {
		Component []struct {
			Group   string `xml:"group,attr"`
			Name    string `xml:"name,attr"`
			Version string `xml:"version,attr"`
		} `xml:"component"`
	} `xml:"components"`
}

func (p *gradleVerificationParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var metadata verificationMetadata
	if err := xml.Unmarshal(content, &metadata); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, comp := range metadata.Components.Component {
		if comp.Group == "" || comp.Name == "" {
			continue
		}

		name := comp.Group + ":" + comp.Name

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: comp.Version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// nebulaLockParser parses dependencies.lock files (Nebula gradle-dependency-lock-plugin).
type nebulaLockParser struct{}

func (p *nebulaLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lockfile map[string]map[string]nebulaLockEntry
	if err := json.Unmarshal(content, &lockfile); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for config, entries := range lockfile {
		isTest := strings.Contains(strings.ToLower(config), "test")

		for name, entry := range entries {
			if entry.Locked == "" || seen[name] {
				continue
			}
			seen[name] = true

			scope := core.Runtime
			if isTest {
				scope = core.Test
			}

			// Direct deps have "requested", transitive have "firstLevelTransitive"
			direct := entry.Requested != ""

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: entry.Locked,
				Scope:   scope,
				Direct:  direct,
			})
		}
	}

	return deps, nil
}

type nebulaLockEntry struct {
	Locked               string   `json:"locked"`
	Requested            string   `json:"requested"`
	FirstLevelTransitive []string `json:"firstLevelTransitive"`
	Project              bool     `json:"project"`
}

// gradleHtmlReportParser parses gradle-html-dependency-report.js files.
type gradleHtmlReportParser struct{}

func (p *gradleHtmlReportParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	// Extract JSON from: window.project = { ... };
	text := string(content)

	// Find the start of the JSON object
	start := strings.Index(text, "window.project = ")
	if start < 0 {
		start = strings.Index(text, "window.project=")
		if start < 0 {
			return nil, &core.ParseError{Filename: filename, Err: errors.New("missing window.project assignment")}
		}
		start += len("window.project=")
	} else {
		start += len("window.project = ")
	}

	// Find the end (last } or };)
	end := strings.LastIndex(text, "}")
	if end < 0 || end < start {
		return nil, &core.ParseError{Filename: filename, Err: errors.New("invalid JSON structure")}
	}

	jsonContent := text[start : end+1]

	var project gradleHtmlProject
	if err := json.Unmarshal([]byte(jsonContent), &project); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, config := range project.Configurations {
		isTest := strings.Contains(strings.ToLower(config.Name), "test")
		collectGradleHtmlDeps(&deps, seen, config.Dependencies, isTest)
	}

	return deps, nil
}

type gradleHtmlProject struct {
	Name           string                  `json:"name"`
	Configurations []gradleHtmlConfig      `json:"configurations"`
}

type gradleHtmlConfig struct {
	Name         string             `json:"name"`
	Dependencies []gradleHtmlDep    `json:"dependencies"`
}

type gradleHtmlDep struct {
	Module   string          `json:"module"`
	Children []gradleHtmlDep `json:"children"`
}

func collectGradleHtmlDeps(deps *[]core.Dependency, seen map[string]bool, htmlDeps []gradleHtmlDep, isTest bool) {
	for _, dep := range htmlDeps {
		// Parse module: "group:artifact:version"
		parts := strings.Split(dep.Module, ":")
		if len(parts) < 3 {
			continue
		}

		name := parts[0] + ":" + parts[1]
		version := parts[2]

		if !seen[name] {
			seen[name] = true

			scope := core.Runtime
			if isTest {
				scope = core.Test
			}

			*deps = append(*deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  false,
			})
		}

		// Recursively collect children
		if len(dep.Children) > 0 {
			collectGradleHtmlDeps(deps, seen, dep.Children, isTest)
		}
	}
}
