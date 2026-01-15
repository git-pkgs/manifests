package conan

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"
)

func init() {
	core.Register("conan", core.Manifest, &conanfileTxtParser{}, core.ExactMatch("conanfile.txt"))
	core.Register("conan", core.Manifest, &conanfilePyParser{}, core.ExactMatch("conanfile.py"))
	core.Register("conan", core.Lockfile, &conanLockParser{}, core.ExactMatch("conan.lock"))
}

// conanfileTxtParser parses conanfile.txt files.
type conanfileTxtParser struct{}

func (p *conanfileTxtParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	section := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Detect sections
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(line[1 : len(line)-1])
			continue
		}

		if section == "requires" || section == "build_requires" {
			name, version := parseConanRef(line)
			if name != "" {
				scope := core.Runtime
				if section == "build_requires" {
					scope = core.Build
				}
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

// conanfilePyParser parses conanfile.py files.
type conanfilePyParser struct{}

var (
	// self.requires("name/version") or self.requires("name/version@user/channel")
	conanRequiresRegex = regexp.MustCompile(`self\.requires\s*\(\s*["']([^"']+)["']`)
	// self.build_requires("name/version")
	conanBuildRequiresRegex = regexp.MustCompile(`self\.build_requires\s*\(\s*["']([^"']+)["']`)
)

func (p *conanfilePyParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)

	for _, match := range conanRequiresRegex.FindAllStringSubmatch(text, -1) {
		name, version := parseConanRef(match[1])
		if name != "" {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	for _, match := range conanBuildRequiresRegex.FindAllStringSubmatch(text, -1) {
		name, version := parseConanRef(match[1])
		if name != "" {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Build,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// conanLockParser parses conan.lock files.
type conanLockParser struct{}

type conanLock struct {
	Requires      []string `json:"requires"`
	BuildRequires []string `json:"build_requires"`
}

func (p *conanLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock conanLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, ref := range lock.Requires {
		name, version := parseConanLockRef(ref)
		if name != "" {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  false,
			})
		}
	}

	for _, ref := range lock.BuildRequires {
		name, version := parseConanLockRef(ref)
		if name != "" {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Build,
				Direct:  false,
			})
		}
	}

	return deps, nil
}

// parseConanRef parses a Conan reference like "name/version" or "name/version@user/channel".
func parseConanRef(ref string) (name, version string) {
	// Remove @user/channel suffix if present
	if idx := strings.Index(ref, "@"); idx > 0 {
		ref = ref[:idx]
	}

	parts := strings.SplitN(ref, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// parseConanLockRef parses a Conan lock reference like "name/version#revision".
func parseConanLockRef(ref string) (name, version string) {
	// Remove #revision suffix if present
	if idx := strings.Index(ref, "#"); idx > 0 {
		ref = ref[:idx]
	}
	return parseConanRef(ref)
}
