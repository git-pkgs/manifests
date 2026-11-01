package git

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

const keyValueParts = 2 // key=value

func init() {
	core.Register("git", core.Manifest, &gitmodulesParser{}, core.ExactMatch(".gitmodules"))
}

// gitmodulesParser parses .gitmodules files.
type gitmodulesParser struct{}

var submoduleHeaderRegex = regexp.MustCompile(`^\[submodule\s+"([^"]+)"\]`)

type submodule struct {
	name   string
	path   string
	url    string
	branch string
}

func (p *gitmodulesParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	var current *submodule
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if match := submoduleHeaderRegex.FindStringSubmatch(line); match != nil {
			if current != nil {
				deps = append(deps, current.dependency())
			}
			current = &submodule{name: match[1]}
			continue
		}

		if current == nil {
			continue
		}

		key, val, ok := parseKeyValue(line)
		if !ok {
			continue
		}
		switch key {
		case "path":
			current.path = val
		case "url":
			current.url = val
		case "branch":
			current.branch = val
		}
	}

	if current != nil {
		deps = append(deps, current.dependency())
	}

	return &core.Result{Dependencies: deps}, nil
}

// dependency identifies the submodule by the host and path of its clone URL.
// The checkout path is where the superproject places it, not what it is, so
// a submodule whose URL is relative or local has no package identity.
func (s *submodule) dependency() core.Dependency {
	dep := core.Dependency{
		Name:   s.path,
		Scope:  core.Runtime,
		Direct: true,
	}
	if dep.Name == "" {
		dep.Name = s.name
	}
	if s.url != "" {
		dep.Source = core.Source{Kind: core.SourceGit, Value: s.url, Branch: s.branch}
	}
	if coordinate := core.GitCoordinate(s.url); coordinate != "" {
		dep.Name = coordinate
	} else {
		dep.NoPURL = true
	}
	return dep
}

func parseKeyValue(line string) (key, value string, ok bool) {
	parts := strings.SplitN(line, "=", keyValueParts)
	if len(parts) != keyValueParts {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}
