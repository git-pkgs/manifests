package rebar

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("hex", core.Lockfile, &rebarLockParser{}, core.ExactMatch("rebar.lock"))
}

// rebarLockParser parses rebar.lock files (Erlang/Elixir).
type rebarLockParser struct{}

var (
	// Match: {<<"pkg_name">>,{pkg,<<"pkg_name">>,<<"version">>},N}
	rebarPkgRegex = regexp.MustCompile(`\{<<"([^"]+)">>,\{pkg,<<"[^"]+">>,<<"([^"]+)">>\},\d+\}`)
)

func (p *rebarLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)

	for _, match := range rebarPkgRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := match[2]

		// Skip if name looks like a hash
		if strings.HasPrefix(name, "sha") || len(name) > 50 {
			continue
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}
