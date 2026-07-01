package luarocks

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("luarocks", core.Manifest, &rockspecParser{}, core.SuffixMatch(".rockspec"))
}

// rockspecParser parses *.rockspec files (Lua).
type rockspecParser struct{}

var (
	// Match dependency strings like "lua >= 5.1" or "luafilesystem >= 1.8.0"
	rockspecDepRegex = regexp.MustCompile(`"([^"]+)"`)
	// Match top-level package = "..."
	rockspecPackageRegex = regexp.MustCompile(`(?m)^\s*package\s*=\s*"([^"]+)"`)
	// Match top-level version = "..."
	rockspecVersionRegex = regexp.MustCompile(`(?m)^\s*version\s*=\s*"([^"]+)"`)
)

func (p *rockspecParser) Parse(filename string, content []byte) (*core.Result, error) {
	text := string(content)

	var selfName, selfVersion string
	if m := rockspecPackageRegex.FindStringSubmatch(text); m != nil {
		selfName = m[1]
	}
	if m := rockspecVersionRegex.FindStringSubmatch(text); m != nil {
		selfVersion = m[1]
	}

	// Find the dependencies section
	depsStart := strings.Index(text, "dependencies")
	if depsStart == -1 {
		return &core.Result{Name: selfName, Version: selfVersion}, nil
	}

	// Find the opening brace
	braceStart := strings.Index(text[depsStart:], "{")
	if braceStart == -1 {
		return &core.Result{Name: selfName, Version: selfVersion}, nil
	}

	// Find matching closing brace
	braceCount := 1
	braceEnd := -1
	start := depsStart + braceStart + 1
	for i := start; i < len(text) && braceCount > 0; i++ {
		switch text[i] {
		case '{':
			braceCount++
		case '}':
			braceCount--
			if braceCount == 0 {
				braceEnd = i
			}
		}
	}

	if braceEnd == -1 {
		return &core.Result{Name: selfName, Version: selfVersion}, nil
	}

	depsSection := text[start:braceEnd]

	var deps []core.Dependency

	for _, match := range rockspecDepRegex.FindAllStringSubmatch(depsSection, -1) {
		depStr := strings.TrimSpace(match[1])
		if depStr == "" {
			continue
		}

		name, version := parseRockspecDep(depStr)
		if name == "" {
			continue
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return &core.Result{Name: selfName, Version: selfVersion, Dependencies: deps}, nil
}

// parseRockspecDep parses a rockspec dependency string like "lua >= 5.1"
func parseRockspecDep(dep string) (name, version string) {
	// Common patterns: "name", "name >= 1.0", "name ~> 1.0", "name == 1.0"
	parts := strings.Fields(dep)
	if len(parts) == 0 {
		return "", ""
	}

	const nameAndOp = 3 // name + operator + version
	name = parts[0]
	if len(parts) >= nameAndOp {
		version = strings.Join(parts[1:], " ")
	} else if len(parts) == 2 { //nolint:mnd // name + bare version
		version = parts[1]
	}

	return name, version
}
