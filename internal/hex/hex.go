package hex

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("hex", core.Manifest, &mixExsParser{}, core.ExactMatch("mix.exs"))
	core.Register("hex", core.Lockfile, &mixLockParser{}, core.ExactMatch("mix.lock"))
}

// mixExsParser parses mix.exs files (Elixir).
type mixExsParser struct{}

var (
	// {:name, "~> 1.0"} or {:name, ">= 1.0"}
	mixDepRegex = regexp.MustCompile(`\{:([a-zA-Z_][a-zA-Z0-9_]*),\s*"([^"]+)"`)
)

func (p *mixExsParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)

	// Find deps function content
	depsStart := strings.Index(text, "defp deps do")
	if depsStart < 0 {
		depsStart = strings.Index(text, "def deps do")
	}
	if depsStart < 0 {
		return deps, nil
	}

	// Find matching end
	section := text[depsStart:]
	depth := 0
	started := false
	end := len(section)
	for i, ch := range section {
		if ch == '[' || ch == '{' {
			depth++
			started = true
		}
		if ch == ']' || ch == '}' {
			depth--
		}
		if started && depth == 0 {
			end = i
			break
		}
	}
	section = section[:end]

	for _, match := range mixDepRegex.FindAllStringSubmatch(section, -1) {
		deps = append(deps, core.Dependency{
			Name:    match[1],
			Version: match[2],
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}

// mixLockParser parses mix.lock files.
type mixLockParser struct{}

var (
	// "package": {:hex, :package, "version", "hash", ...}
	mixLockRegex = regexp.MustCompile(`"([^"]+)":\s*\{:hex,\s*:([^,]+),\s*"([^"]+)",\s*"([^"]+)"`)
)

func (p *mixLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)

	for _, match := range mixLockRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := match[3]
		hash := match[4]

		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:   core.Runtime,
			Integrity: "sha256-" + hash,
			Direct:    false,
		})
	}

	return deps, nil
}
