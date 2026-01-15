package julia

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"

	"github.com/BurntSushi/toml"
)

func init() {
	core.Register("julia", core.Manifest, &juliaProjectParser{}, core.ExactMatch("Project.toml"))
	core.Register("julia", core.Lockfile, &juliaManifestParser{}, core.ExactMatch("Manifest.toml"))
	core.Register("julia", core.Manifest, &juliaRequireParser{}, core.ExactMatch("REQUIRE"))
}

// extractJuliaDepName extracts name from [[deps.Name]]
func extractJuliaDepName(line string) (string, bool) {
	if !strings.HasPrefix(line, "[[deps.") {
		return "", false
	}
	end := strings.IndexByte(line, ']')
	if end < 7 {
		return "", false
	}
	return line[7:end], true
}

// juliaProjectParser parses Julia Project.toml files.
type juliaProjectParser struct{}

type juliaProject struct {
	Deps   map[string]string `toml:"deps"`
	Compat map[string]string `toml:"compat"`
}

func (p *juliaProjectParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var project juliaProject
	if err := toml.Unmarshal(content, &project); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name := range project.Deps {
		version := ""
		if v, ok := project.Compat[name]; ok {
			version = v
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

// juliaManifestParser parses Julia Manifest.toml files using regex for speed.
type juliaManifestParser struct{}

func (p *juliaManifestParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	var currentName, currentVersion, currentGitSHA string

	core.ForEachLine(text, func(line string) bool {
		// Start of a dep block [[deps.Name]]
		if name, ok := extractJuliaDepName(line); ok {
			// Save previous dep
			if currentName != "" {
				version := currentVersion
				if version == "" {
					version = currentGitSHA
				}
				deps = append(deps, core.Dependency{
					Name:    currentName,
					Version: version,
					Scope:   core.Runtime,
					Direct:  false,
				})
			}
			currentName = name
			currentVersion = ""
			currentGitSHA = ""
			return true
		}

		if currentName != "" {
			if v, ok := core.ExtractQuotedValue(line, "version = "); ok {
				currentVersion = v
			} else if v, ok := core.ExtractQuotedValue(line, "git-tree-sha1 = "); ok {
				currentGitSHA = v
			}
		}
		return true
	})

	// Don't forget the last dep
	if currentName != "" {
		version := currentVersion
		if version == "" {
			version = currentGitSHA
		}
		deps = append(deps, core.Dependency{
			Name:    currentName,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// juliaRequireParser parses legacy Julia REQUIRE files.
type juliaRequireParser struct{}

func (p *juliaRequireParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency

	core.ForEachLine(string(content), func(line string) bool {
		// Skip empty lines
		if len(line) == 0 {
			return true
		}

		// Skip comments
		if line[0] == '#' {
			return true
		}

		// Skip lines starting with whitespace or dash (continuation/list items)
		if line[0] == ' ' || line[0] == '\t' || line[0] == '-' {
			return true
		}

		// Handle platform-specific deps: @osx Homebrew
		if line[0] == '@' {
			// Skip platform marker and get the rest
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				line = strings.Join(parts[1:], " ")
			} else {
				return true
			}
		}

		// Parse: PackageName [version [max_version]]
		parts := strings.Fields(line)
		if len(parts) == 0 {
			return true
		}

		name := parts[0]
		version := ""
		if len(parts) >= 2 {
			// Version can be single (0.3.4) or range (0.12 0.15)
			if len(parts) == 2 {
				version = parts[1]
			} else {
				// Range: min max
				version = parts[1] + " " + parts[2]
			}
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
		return true
	})

	return deps, nil
}
