package maven

import (
	"encoding/xml"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("maven", core.Manifest, &pomXMLParser{}, core.ExactMatch("pom.xml"))

	// maven-resolved-dependencies.txt - lockfile (mvn dependency:list output)
	core.Register("maven", core.Lockfile, &mavenResolvedDepsParser{}, core.ExactMatch("maven-resolved-dependencies.txt"))
}

// pomXMLParser parses pom.xml files.
type pomXMLParser struct{}

type pomXML struct {
	Dependencies struct {
		Dependency []pomDependency `xml:"dependency"`
	} `xml:"dependencies"`
}

type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
	Scope      string `xml:"scope"`
	Optional   string `xml:"optional"`
}

func (p *pomXMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var pom pomXML
	if err := xml.Unmarshal(content, &pom); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, dep := range pom.Dependencies.Dependency {
		// Skip dependencies with property references as they can't be resolved
		if strings.Contains(dep.GroupID, "${") && strings.Contains(dep.ArtifactID, "${") {
			continue
		}

		name := dep.GroupID + ":" + dep.ArtifactID
		scope := core.Runtime

		switch strings.ToLower(dep.Scope) {
		case "test":
			scope = core.Test
		case "provided", "compile":
			scope = core.Runtime
		case "runtime":
			scope = core.Runtime
		}

		if strings.ToLower(dep.Optional) == "true" {
			scope = core.Optional
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: dep.Version,
			Scope:   scope,
			Direct:  true,
		})
	}

	return deps, nil
}

// mavenResolvedDepsParser parses maven-resolved-dependencies.txt files (mvn dependency:list output).
type mavenResolvedDepsParser struct{}

// Match lines like: org.group:artifact:jar:version:scope
// Format: group:artifact:type:version:scope or group:artifact:type:classifier:version:scope
var mavenResolvedDepRegex = regexp.MustCompile(`^\s*([a-zA-Z0-9._-]+):([a-zA-Z0-9._-]+):[a-z-]+:([^:]+):([a-z]+)`)

func (p *mavenResolvedDepsParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[string]bool)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Strip ANSI escape codes
		line = stripANSI(line)

		if match := mavenResolvedDepRegex.FindStringSubmatch(line); match != nil {
			groupID := match[1]
			artifactID := match[2]
			version := match[3]
			scopeStr := match[4]

			name := groupID + ":" + artifactID

			if seen[name] {
				continue
			}
			seen[name] = true

			scope := core.Runtime
			switch scopeStr {
			case "test":
				scope = core.Test
			case "provided":
				scope = core.Runtime
			case "runtime":
				scope = core.Runtime
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

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	ansiEscapeRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiEscapeRegex.ReplaceAllString(s, "")
}
