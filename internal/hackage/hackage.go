package hackage

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("hackage", core.Manifest, &cabalParser{}, core.SuffixMatch(".cabal"))
	core.Register("hackage", core.Lockfile, &stackLockParser{}, core.ExactMatch("stack.yaml.lock"))
	core.Register("hackage", core.Lockfile, &cabalConfigParser{}, core.ExactMatch("cabal.config"))
	core.Register("hackage", core.Lockfile, &cabalFreezeParser{}, core.ExactMatch("cabal.project.freeze"))
}

// cabalParser parses *.cabal files.
type cabalParser struct{}

var (
	// build-depends: or build-tool-depends:
	cabalBuildDependsRegex = regexp.MustCompile(`(?i)^\s*build-depends:\s*$|(?i)^\s*build-tool-depends:\s*$`)
	// Package name with optional version constraint
	cabalDepRegex = regexp.MustCompile(`^\s*,?\s*([a-zA-Z][a-zA-Z0-9-]*)\s*(.*)$`)
)

func (p *cabalParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")
	seen := make(map[string]bool)

	inBuildDepends := false

	for _, line := range lines {
		// Check for build-depends section
		if cabalBuildDependsRegex.MatchString(line) {
			inBuildDepends = true
			continue
		}

		// Check for inline build-depends
		if strings.Contains(strings.ToLower(line), "build-depends:") {
			inBuildDepends = true
			// Parse the rest of the line
			idx := strings.Index(strings.ToLower(line), "build-depends:")
			line = line[idx+14:]
		}

		if !inBuildDepends {
			continue
		}

		// End of section if line starts with non-space (new section)
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(trimmed, ",") {
			inBuildDepends = false
			continue
		}

		// Parse dependencies (may be comma-separated)
		parts := strings.Split(line, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}

			if match := cabalDepRegex.FindStringSubmatch(part); match != nil {
				name := match[1]
				version := strings.TrimSpace(match[2])

				if seen[name] {
					continue
				}
				seen[name] = true

				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
					Scope:   core.Runtime,
					Direct:  true,
				})
			}
		}
	}

	return deps, nil
}

// stackLockParser parses stack.yaml.lock files.
type stackLockParser struct{}

type stackLock struct {
	Packages []stackLockPackage `yaml:"packages"`
}

type stackLockPackage struct {
	Completed struct {
		Hackage string `yaml:"hackage"`
	} `yaml:"completed"`
}

func (p *stackLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock stackLock
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, pkg := range lock.Packages {
		hackage := pkg.Completed.Hackage
		if hackage == "" {
			continue
		}

		// Format: name-version@sha256:hash,size
		name, version := parseHackageRef(hackage)
		if name == "" {
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

// parseHackageRef parses a hackage reference like "name-1.2.3@sha256:..."
func parseHackageRef(ref string) (name, version string) {
	// Remove @sha256:... suffix
	if idx := strings.Index(ref, "@"); idx > 0 {
		ref = ref[:idx]
	}

	// Split name and version (version starts at last hyphen followed by digit)
	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == '-' && i < len(ref)-1 && ref[i+1] >= '0' && ref[i+1] <= '9' {
			return ref[:i], ref[i+1:]
		}
	}
	return ref, ""
}

// cabalConfigParser parses cabal.config files (freeze files).
type cabalConfigParser struct{}

var (
	// Match: pkg ==version or pkg ==version,
	cabalConstraintRegex = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9-]*)\s*==\s*([0-9][0-9.]*[0-9])`)
)

func (p *cabalConfigParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	for _, match := range cabalConstraintRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := match[2]

		if seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// cabalFreezeParser parses cabal.project.freeze files.
type cabalFreezeParser struct{}

var (
	// Match: any.pkg ==version or any.pkg ==version,
	cabalProjectFreezeRegex = regexp.MustCompile(`any\.([a-zA-Z][a-zA-Z0-9-]*)\s*==\s*([0-9][0-9.]*[0-9])`)
)

func (p *cabalFreezeParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	for _, match := range cabalProjectFreezeRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := match[2]

		if seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}
