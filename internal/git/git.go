package git

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("git", core.Manifest, &gitmodulesParser{}, core.ExactMatch(".gitmodules"))
}

// gitmodulesParser parses .gitmodules files.
type gitmodulesParser struct{}

var submoduleHeaderRegex = regexp.MustCompile(`^\[submodule\s+"([^"]+)"\]`)

func (p *gitmodulesParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	var current *core.Dependency
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if match := submoduleHeaderRegex.FindStringSubmatch(line); match != nil {
			if current != nil {
				deps = append(deps, *current)
			}
			current = &core.Dependency{
				Name:   match[1],
				Scope:  core.Runtime,
				Direct: true,
			}
			continue
		}

		if current == nil {
			continue
		}

		if strings.HasPrefix(line, "path") {
			if _, val, ok := parseKeyValue(line); ok {
				current.Name = val
			}
		} else if strings.HasPrefix(line, "url") {
			if _, val, ok := parseKeyValue(line); ok {
				current.RegistryURL = val
			}
		}
	}

	if current != nil {
		deps = append(deps, *current)
	}

	return deps, nil
}

func parseKeyValue(line string) (string, string, bool) {
	parts := strings.SplitN(line, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}
