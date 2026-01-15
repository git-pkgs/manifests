package maven

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/xml"
	"strings"
)

func init() {
	core.Register("maven", core.Manifest, &pomXMLParser{}, core.ExactMatch("pom.xml"))
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
