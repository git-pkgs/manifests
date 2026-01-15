package cpan

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("cpan", core.Manifest, &cpanfileParser{}, core.ExactMatch("cpanfile"))
	core.Register("cpan", core.Lockfile, &cpanfileSnapshotParser{}, core.ExactMatch("cpanfile.snapshot"))
	core.Register("cpan", core.Manifest, &makefilePLParser{}, core.ExactMatch("Makefile.PL"))
	core.Register("cpan", core.Manifest, &buildPLParser{}, core.ExactMatch("Build.PL"))
	core.Register("cpan", core.Manifest, &distIniParser{}, core.ExactMatch("dist.ini"))
	core.Register("cpan", core.Manifest, &metaJSONParser{}, core.ExactMatch("META.json"))
	core.Register("cpan", core.Manifest, &metaYMLParser{}, core.ExactMatch("META.yml"))
}

// cpanfileParser parses cpanfile files.
type cpanfileParser struct{}

// extractCpanRequires parses requires 'Name', 'version'; or requires 'Name';
func extractCpanRequires(line string) (name, version string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "requires ") {
		return "", "", false
	}

	rest := trimmed[9:] // len("requires ")

	// Find first quote
	start := strings.IndexAny(rest, "'\"")
	if start < 0 {
		return "", "", false
	}
	quote := rest[start]

	// Find end of name
	end := strings.IndexByte(rest[start+1:], quote)
	if end < 0 {
		return "", "", false
	}
	name = rest[start+1 : start+1+end]

	// Look for version after comma
	rest = rest[start+1+end+1:]
	if idx := strings.IndexByte(rest, ','); idx >= 0 {
		rest = rest[idx+1:]
		vstart := strings.IndexAny(rest, "'\"")
		if vstart >= 0 {
			vquote := rest[vstart]
			vend := strings.IndexByte(rest[vstart+1:], vquote)
			if vend >= 0 {
				version = rest[vstart+1 : vstart+1+vend]
			}
		}
	}

	return name, version, true
}

func (p *cpanfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	core.ForEachLine(text, func(line string) bool {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' {
			return true
		}

		if name, version, ok := extractCpanRequires(line); ok {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
		return true
	})

	return deps, nil
}

// cpanfileSnapshotParser parses cpanfile.snapshot files.
type cpanfileSnapshotParser struct{}

// extractCpanProvides parses "      Module::Name version" lines
func extractCpanProvides(line string) (name, version string, ok bool) {
	// Must start with 6 spaces
	if len(line) < 7 || line[0] != ' ' || line[1] != ' ' || line[2] != ' ' ||
		line[3] != ' ' || line[4] != ' ' || line[5] != ' ' || line[6] == ' ' {
		return "", "", false
	}

	// Parse "Module::Name version"
	rest := line[6:]
	idx := strings.IndexByte(rest, ' ')
	if idx < 0 {
		return "", "", false
	}

	name = rest[:idx]
	version = strings.TrimSpace(rest[idx+1:])

	if version == "undef" {
		version = ""
	}

	return name, version, true
}

func (p *cpanfileSnapshotParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	inDistribution := false
	inProvides := false

	core.ForEachLine(text, func(line string) bool {
		if line == "" || (len(line) > 0 && line[0] == '#') {
			return true
		}

		if line == "DISTRIBUTIONS" {
			inDistribution = true
			return true
		}

		if !inDistribution {
			return true
		}

		// Distribution entry (2 spaces, not 4)
		if len(line) >= 2 && line[0] == ' ' && line[1] == ' ' && (len(line) < 3 || line[2] != ' ') {
			inProvides = false
			return true
		}

		// Check for provides: section
		if strings.TrimSpace(line) == "provides:" {
			inProvides = true
			return true
		}

		// Other subsections (4 spaces, not 6) end provides
		if len(line) >= 4 && line[0] == ' ' && line[1] == ' ' && line[2] == ' ' && line[3] == ' ' &&
			(len(line) < 5 || line[4] != ' ') {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 && trimmed[len(trimmed)-1] == ':' {
				inProvides = false
			}
			return true
		}

		if inProvides {
			if name, version, ok := extractCpanProvides(line); ok {
				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
					Scope:   core.Runtime,
					Direct:  false,
				})
			}
		}
		return true
	})

	return deps, nil
}

// makefilePLParser parses Makefile.PL files.
type makefilePLParser struct{}

var (
	// Match: 'Module::Name' => 'version' or 'Module::Name' => version
	perlDepRegex = regexp.MustCompile(`['"]([A-Za-z][A-Za-z0-9:_]*)['"]?\s*=>\s*['"]?([^'",\s}]*)['"]?`)
)

func (p *makefilePLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	// Find PREREQ_PM, BUILD_REQUIRES, TEST_REQUIRES, CONFIGURE_REQUIRES sections
	sections := map[string]core.Scope{
		"PREREQ_PM":          core.Runtime,
		"BUILD_REQUIRES":     core.Build,
		"TEST_REQUIRES":      core.Test,
		"CONFIGURE_REQUIRES": core.Build,
	}

	for section, scope := range sections {
		deps = append(deps, parsePerlHashSection(text, section, scope, seen)...)
	}

	return deps, nil
}

// buildPLParser parses Build.PL files.
type buildPLParser struct{}

func (p *buildPLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	// Find requires, build_requires, test_requires, configure_requires sections
	sections := map[string]core.Scope{
		"requires":           core.Runtime,
		"build_requires":     core.Build,
		"test_requires":      core.Test,
		"configure_requires": core.Build,
	}

	for section, scope := range sections {
		deps = append(deps, parsePerlHashSection(text, section, scope, seen)...)
	}

	return deps, nil
}

// isAlphaUnderscore checks if a byte is a letter or underscore
func isAlphaUnderscore(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

// parsePerlHashSection extracts deps from a Perl hash section like: section => { 'Mod' => 'ver', ... }
func parsePerlHashSection(text, section string, scope core.Scope, seen map[string]bool) []core.Dependency {
	var deps []core.Dependency

	// Find the section - must be at word boundary (not part of longer name like configure_requires)
	searchStart := 0
	idx := -1
	for {
		pos := strings.Index(text[searchStart:], section)
		if pos == -1 {
			break
		}
		pos += searchStart

		// Check if this is a word boundary match
		validStart := pos == 0 || !isAlphaUnderscore(text[pos-1])
		validEnd := pos+len(section) >= len(text) || !isAlphaUnderscore(text[pos+len(section)])

		if validStart && validEnd {
			idx = pos
			break
		}
		searchStart = pos + 1
	}

	if idx == -1 {
		return nil
	}

	// Find the opening brace
	braceStart := strings.Index(text[idx:], "{")
	if braceStart == -1 {
		return nil
	}

	// Find matching closing brace
	start := idx + braceStart + 1
	braceCount := 1
	braceEnd := -1
	for i := start; i < len(text) && braceCount > 0; i++ {
		if text[i] == '{' {
			braceCount++
		} else if text[i] == '}' {
			braceCount--
			if braceCount == 0 {
				braceEnd = i
			}
		}
	}

	if braceEnd == -1 {
		return nil
	}

	sectionContent := text[start:braceEnd]

	for _, match := range perlDepRegex.FindAllStringSubmatch(sectionContent, -1) {
		name := match[1]
		version := match[2]

		// Skip perl itself and common non-module entries
		if name == "perl" || seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   scope,
			Direct:  true,
		})
	}

	return deps
}

// distIniParser parses dist.ini files (Dist::Zilla).
type distIniParser struct{}

func (p *distIniParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	// dist.ini uses [AutoPrereqs] to auto-detect dependencies
	// We can't extract deps directly without running Perl
	// Return empty for now - this is mainly for file identification
	return nil, nil
}

// metaJSONParser parses META.json files.
type metaJSONParser struct{}

type metaJSON struct {
	Prereqs map[string]map[string]map[string]string `json:"prereqs"`
}

func (p *metaJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var meta metaJSON
	if err := json.Unmarshal(content, &meta); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	phaseScopes := map[string]core.Scope{
		"runtime":   core.Runtime,
		"build":     core.Build,
		"test":      core.Test,
		"configure": core.Build,
		"develop":   core.Development,
	}

	for phase, requirements := range meta.Prereqs {
		scope := phaseScopes[phase]
		if scope == "" {
			scope = core.Runtime
		}

		for _, mods := range requirements {
			for name, version := range mods {
				if name == "perl" || seen[name] {
					continue
				}
				seen[name] = true

				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
					Scope:   scope,
					Direct:  true,
				})
			}
		}
	}

	return deps, nil
}

// metaYMLParser parses META.yml files.
type metaYMLParser struct{}

type metaYML struct {
	Requires          map[string]any `yaml:"requires"`
	BuildRequires     map[string]any `yaml:"build_requires"`
	ConfigureRequires map[string]any `yaml:"configure_requires"`
	TestRequires      map[string]any `yaml:"test_requires"`
	Recommends        map[string]any `yaml:"recommends"`
}

func (p *metaYMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var meta metaYML
	if err := yaml.Unmarshal(content, &meta); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	sections := map[*map[string]any]core.Scope{
		&meta.Requires:          core.Runtime,
		&meta.BuildRequires:     core.Build,
		&meta.ConfigureRequires: core.Build,
		&meta.TestRequires:      core.Test,
		&meta.Recommends:        core.Optional,
	}

	for mods, scope := range sections {
		if mods == nil || *mods == nil {
			continue
		}
		for name, ver := range *mods {
			if name == "perl" || seen[name] {
				continue
			}
			seen[name] = true

			version := ""
			switch v := ver.(type) {
			case string:
				version = v
			case int:
				version = "0"
			case float64:
				version = "0"
			}

			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
		}
	}

	return deps, nil
}
