package maven

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/xml"
	"strings"
)

func init() {
	core.Register("maven", core.Manifest, &ivyXMLParser{}, core.ExactMatch("ivy.xml"))
}

// ivyXMLParser parses ivy.xml files.
type ivyXMLParser struct{}

type ivyModule struct {
	Dependencies struct {
		Deps []ivyDep `xml:"dependency"`
	} `xml:"dependencies"`
}

type ivyDep struct {
	Org  string `xml:"org,attr"`
	Name string `xml:"name,attr"`
	Rev  string `xml:"rev,attr"`
	Conf string `xml:"conf,attr"`
}

func (p *ivyXMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var module ivyModule
	if err := xml.Unmarshal(content, &module); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, dep := range module.Dependencies.Deps {
		name := dep.Org + ":" + dep.Name
		if seen[name] {
			continue
		}
		seen[name] = true

		scope := core.Runtime
		// Check configurations before the -> (the local conf)
		localConf := dep.Conf
		if idx := strings.Index(localConf, "->"); idx > 0 {
			localConf = localConf[:idx]
		}
		// If local conf is only "test" (doesn't include "default"), it's test scope
		if strings.Contains(localConf, "test") && !strings.Contains(localConf, "default") {
			scope = core.Test
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: dep.Rev,
			Scope:   scope,
			Direct:  true,
		})
	}

	return deps, nil
}
