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
	// Match: {<<"pkg_name">>, <<"sha256">>}
	rebarHashRegex = regexp.MustCompile(`\{<<"([^"]+)">>,\s*<<"([^"]+)">>\}`)
)

func (p *rebarLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	text := string(content)
	hashes := rebarPackageHashes(text)

	for _, match := range rebarPkgRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := match[2]

		// Skip if name looks like a hash
		if strings.HasPrefix(name, "sha") || len(name) > 50 {
			continue
		}

		integrity := ""
		if hash := hashes[name]; hash != "" {
			integrity = "sha256-" + hash
		}

		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:     core.Runtime,
			Integrity: integrity,
			Direct:    false,
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

func rebarPackageHashes(text string) map[string]string {
	hashes := rebarHashSection(text, "pkg_hash")
	for name, hash := range rebarHashSection(text, "pkg_hash_ext") {
		// pkg_hash_ext is the current outer tarball checksum. Prefer it over
		// the deprecated inner checksum stored in pkg_hash.
		hashes[name] = hash
	}
	return hashes
}

func rebarHashSection(text, name string) map[string]string {
	hashes := make(map[string]string)
	marker := "{" + name + ",["
	start := strings.Index(text, marker)
	if start < 0 {
		return hashes
	}

	section := text[start+len(marker):]
	end := strings.Index(section, "]}")
	if end < 0 {
		return hashes
	}

	for _, match := range rebarHashRegex.FindAllStringSubmatch(section[:end], -1) {
		hashes[match[1]] = match[2]
	}
	return hashes
}
