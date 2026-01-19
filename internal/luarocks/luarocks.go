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
)

func (p *rockspecParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)

	// Find the dependencies section
	depsStart := strings.Index(text, "dependencies")
	if depsStart == -1 {
		return nil, nil
	}

	// Find the opening brace
	braceStart := strings.Index(text[depsStart:], "{")
	if braceStart == -1 {
		return nil, nil
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
		return nil, nil
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

	return deps, nil
}

// parseRockspecDep parses a rockspec dependency string like "lua >= 5.1"
func parseRockspecDep(dep string) (name, version string) {
	// Common patterns: "name", "name >= 1.0", "name ~> 1.0", "name == 1.0"
	parts := strings.Fields(dep)
	if len(parts) == 0 {
		return "", ""
	}

	name = parts[0]
	if len(parts) >= 3 {
		version = strings.Join(parts[1:], " ")
	} else if len(parts) == 2 {
		version = parts[1]
	}

	return name, version
}
