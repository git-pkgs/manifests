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
	// app: :name inside def project
	mixAppRegex = regexp.MustCompile(`\bapp:\s*:([a-zA-Z_][a-zA-Z0-9_]*)`)
	// version: "0.0.1" inside def project
	mixVersionRegex = regexp.MustCompile(`\bversion:\s*"([^"]+)"`)
)

func (p *mixExsParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	text := string(content)

	var selfName, selfVersion string
	if projectStart := strings.Index(text, "def project"); projectStart >= 0 {
		section := extractMixBlock(text[projectStart:])
		if m := mixAppRegex.FindStringSubmatch(section); m != nil {
			selfName = m[1]
		}
		if m := mixVersionRegex.FindStringSubmatch(section); m != nil {
			selfVersion = m[1]
		}
	}

	// Find deps function content
	depsStart := strings.Index(text, "defp deps do")
	if depsStart < 0 {
		depsStart = strings.Index(text, "def deps do")
	}
	if depsStart < 0 {
		return &core.Result{Name: selfName, Version: selfVersion, Dependencies: deps}, nil
	}

	section := extractMixBlock(text[depsStart:])
	for _, match := range mixDepRegex.FindAllStringSubmatch(section, -1) {
		deps = append(deps, core.Dependency{
			Name:    match[1],
			Version: match[2],
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return &core.Result{Name: selfName, Version: selfVersion, Dependencies: deps}, nil
}

// extractMixBlock returns the bracket-delimited body starting at text.
func extractMixBlock(text string) string {
	depth := 0
	started := false
	for i, ch := range text {
		if ch == '[' || ch == '{' {
			depth++
			started = true
		}
		if ch == ']' || ch == '}' {
			depth--
		}
		if started && depth == 0 {
			return text[:i]
		}
	}
	return text
}

// mixLockParser parses mix.lock files.
type mixLockParser struct{}

var (
	// "package": {:hex, :package, "version", "hash", ...}
	mixLockRegex = regexp.MustCompile(`"([^"]+)":\s*\{:hex,\s*:([^,]+),\s*"([^"]+)",\s*"([^"]+)"`)
)

func (p *mixLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	text := string(content)

	for _, match := range mixLockRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := match[3]
		hash := match[4]

		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:     core.Runtime,
			Integrity: "sha256-" + hash,
			Direct:    false,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}
