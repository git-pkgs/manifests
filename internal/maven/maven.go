package maven

import (
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"github.com/git-pkgs/pom"
)

const (
	scopeTest     = "test"
	scopeProvided = "provided"
)

func init() {
	core.Register("maven", core.Manifest, &pomXMLParser{}, core.ExactMatch("pom.xml"))

	// maven-resolved-dependencies.txt - lockfile (mvn dependency:list output)
	core.Register("maven", core.Lockfile, &mavenResolvedDepsParser{}, core.ExactMatch("maven-resolved-dependencies.txt"))

	// maven.graph.json - lockfile (mvn dependency:tree -DoutputType=json output)
	core.Register("maven", core.Lockfile, &mavenGraphJSONParser{}, core.ExactMatch("maven.graph.json"))
}

// pomXMLParser parses pom.xml files. It computes a local-only effective
// POM: parents reachable via <relativePath> on disk are merged so that
// ${project.version} and properties defined in a multi-module root
// resolve, but nothing is fetched over the network. Anything that would
// need a remote parent or BOM is left as-is and the dependency keeps its
// raw ${...} version.
type pomXMLParser struct{}

func (p *pomXMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	root, err := pom.ParsePOM(content)
	if err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	fetcher := pom.NewLocalFetcherFrom(root, filepath.Dir(filename))
	ep, err := pom.NewResolver(fetcher).ResolvePOM(context.Background(), root, pom.Options{})
	if err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	deps := make([]core.Dependency, 0, len(ep.Dependencies))
	for _, d := range ep.Dependencies {
		if strings.Contains(d.GroupID, "${") && strings.Contains(d.ArtifactID, "${") {
			continue
		}
		deps = append(deps, core.Dependency{
			Name:    d.GroupID + ":" + d.ArtifactID,
			Version: d.Version,
			Scope:   mapScope(d.Scope, d.Optional),
			Direct:  true,
		})
	}
	return deps, nil
}

func mapScope(scope string, optional bool) core.Scope {
	if optional {
		return core.Optional
	}
	switch strings.ToLower(scope) {
	case scopeTest:
		return core.Test
	default:
		return core.Runtime
	}
}

// mavenResolvedDepsParser parses maven-resolved-dependencies.txt files (mvn dependency:list output).
type mavenResolvedDepsParser struct{}

// Match lines like: org.group:artifact:jar:version:scope
// Format: group:artifact:type:version:scope or group:artifact:type:classifier:version:scope
var mavenResolvedDepRegex = regexp.MustCompile(`^\s*([a-zA-Z0-9._-]+):([a-zA-Z0-9._-]+):[a-z-]+:([^:]+):([a-z]+)`)

var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
			case scopeTest:
				scope = core.Test
			case scopeProvided:
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
	return ansiEscapeRegex.ReplaceAllString(s, "")
}

// mavenGraphJSONParser parses maven.graph.json files (mvn dependency:tree -DoutputType=json output).
type mavenGraphJSONParser struct{}

type mavenGraphNode struct {
	GroupID    string           `json:"groupId"`
	ArtifactID string           `json:"artifactId"`
	Version    string           `json:"version"`
	Scope      string           `json:"scope"`
	Children   []mavenGraphNode `json:"children"`
}

func (p *mavenGraphJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var root mavenGraphNode
	if err := json.Unmarshal(content, &root); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	// Collect all children (skip the root which is the project itself)
	collectMavenGraphDeps(&deps, seen, root.Children)

	return deps, nil
}

func collectMavenGraphDeps(deps *[]core.Dependency, seen map[string]bool, nodes []mavenGraphNode) {
	for _, node := range nodes {
		name := node.GroupID + ":" + node.ArtifactID

		if !seen[name] {
			seen[name] = true

			scope := core.Runtime
			switch strings.ToLower(node.Scope) {
			case scopeTest:
				scope = core.Test
			case scopeProvided:
				scope = core.Optional
			}

			*deps = append(*deps, core.Dependency{
				Name:    name,
				Version: node.Version,
				Scope:   scope,
				Direct:  false,
			})
		}

		// Recursively collect children
		if len(node.Children) > 0 {
			collectMavenGraphDeps(deps, seen, node.Children)
		}
	}
}
