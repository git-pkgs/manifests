package carthage

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("carthage", core.Manifest, &cartfileParser{}, core.ExactMatch("Cartfile", "Cartfile.private"))
	core.Register("carthage", core.Lockfile, &cartfileResolvedParser{}, core.ExactMatch("Cartfile.resolved"))
}

// cartfileParser parses Cartfile manifest files.
type cartfileParser struct{}

var (
	// github "owner/repo" >= 1.0 or github "owner/repo" ~> 1.0 or github "owner/repo"
	cartfileGithubRegex = regexp.MustCompile(`^\s*github\s+"([^"]+)"(?:\s+(>=|~>|==|"[^"]*")?\s*(.*))?`)
	// git "url" "branch"
	cartfileGitRegex = regexp.MustCompile(`^\s*git\s+"([^"]+)"(?:\s+"([^"]*)")?`)
)

func (p *cartfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Remove comments
		if idx := strings.Index(line, "#"); idx >= 0 {
			line = line[:idx]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if match := cartfileGithubRegex.FindStringSubmatch(line); match != nil {
			name := match[1]
			version := ""
			if match[2] != "" {
				op := match[2]
				ver := strings.TrimSpace(match[3])
				// Handle quoted branch/tag vs version constraint
				if strings.HasPrefix(op, "\"") {
					// It's a branch/tag reference
					version = strings.Trim(op, "\"")
				} else {
					version = op + ver
				}
			}
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		} else if match := cartfileGitRegex.FindStringSubmatch(line); match != nil {
			name := match[1]
			version := ""
			if len(match) > 2 {
				version = match[2]
			}
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// cartfileResolvedParser parses Cartfile.resolved lockfiles.
type cartfileResolvedParser struct{}

var (
	// github "owner/repo" "version"
	cartfileResolvedRegex = regexp.MustCompile(`^\s*(github|git)\s+"([^"]+)"\s+"([^"]+)"`)
)

func (p *cartfileResolvedParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if match := cartfileResolvedRegex.FindStringSubmatch(line); match != nil {
			name := match[2]
			version := match[3]
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  false, // Resolved file doesn't distinguish direct/transitive
			})
		}
	}

	return deps, nil
}
