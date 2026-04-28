package lake

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("lean", core.Manifest, &lakefileTomlParser{}, core.ExactMatch("lakefile.toml"))
	core.Register("lean", core.Manifest, &lakefileLeanParser{}, core.ExactMatch("lakefile.lean"))
	core.Register("lean", core.Lockfile, &lakeManifestParser{}, core.ExactMatch("lake-manifest.json"))
}

func scopedName(scope, name string) string {
	if scope != "" {
		return scope + "/" + name
	}
	return name
}

// lakefileTomlParser parses lakefile.toml files.
type lakefileTomlParser struct{}

type lakeRequire struct {
	Name    string         `toml:"name"`
	Scope   string         `toml:"scope"`
	Version string         `toml:"version"`
	Git     string         `toml:"git"`
	Rev     string         `toml:"rev"`
	Path    string         `toml:"path"`
	Source  map[string]any `toml:"source"`
}

type lakefileToml struct {
	Name    string        `toml:"name"`
	Require []lakeRequire `toml:"require"`
}

func (p *lakefileTomlParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lake lakefileToml
	if err := toml.Unmarshal(content, &lake); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, r := range lake.Require {
		git, rev, path := r.Git, r.Rev, r.Path
		if r.Source != nil {
			if v, ok := r.Source["git"].(string); ok {
				git = v
			}
			if v, ok := r.Source["url"].(string); ok {
				git = v
			}
			if v, ok := r.Source["rev"].(string); ok {
				rev = v
			}
			if v, ok := r.Source["path"].(string); ok {
				path = v
			}
		}

		if path != "" && git == "" {
			continue
		}

		version := r.Version
		if version == "" {
			version = rev
		}

		deps = append(deps, core.Dependency{
			Name:        scopedName(r.Scope, r.Name),
			Version:     version,
			Scope:       core.Runtime,
			Direct:      true,
			RegistryURL: git,
		})
	}

	return deps, nil
}

// lakefileLeanParser parses lakefile.lean files using regex.
type lakefileLeanParser struct{}

const (
	groupScope     = 1
	groupNameStr   = 2
	groupNameGuill = 3
	groupNameIdent = 4
	groupVersion   = 5
	groupGitURL    = 6
	groupGitRev    = 7
	groupPath      = 8
)

var lakeRequireRegex = regexp.MustCompile(
	`require\s+` +
		`(?:"([^"]+)"\s*/\s*)?` +
		`(?:"([^"]+)"|«([^»]+)»|([A-Za-z_][A-Za-z0-9_]*))` +
		`(?:\s*@\s*(?:git\s+)?"([^"]+)")?` +
		`(?:\s*from\s+(?:git\s+"([^"]+)"(?:\s*@\s*"([^"]+)")?|"([^"]+)"))?`,
)

func (p *lakefileLeanParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := stripLeanLineComments(string(content))

	var deps []core.Dependency
	for _, m := range lakeRequireRegex.FindAllStringSubmatch(text, -1) {
		name := m[groupNameStr]
		if name == "" {
			name = m[groupNameGuill]
		}
		if name == "" {
			name = m[groupNameIdent]
		}
		if name == "" {
			continue
		}

		if m[groupPath] != "" && m[groupGitURL] == "" {
			continue
		}

		version := m[groupVersion]
		if version == "" {
			version = m[groupGitRev]
		}

		deps = append(deps, core.Dependency{
			Name:        scopedName(m[groupScope], name),
			Version:     version,
			Scope:       core.Runtime,
			Direct:      true,
			RegistryURL: m[groupGitURL],
		})
	}

	return deps, nil
}

func stripLeanLineComments(text string) string {
	var b strings.Builder
	b.Grow(len(text))
	core.ForEachLine(text, func(line string) bool {
		if i := strings.Index(line, "--"); i >= 0 && !strings.Contains(line[:i], `"`) {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
		return true
	})
	return b.String()
}

// lakeManifestParser parses lake-manifest.json lockfiles.
type lakeManifestParser struct{}

type lakeManifestPackage struct {
	Name      string `json:"name"`
	Scope     string `json:"scope"`
	URL       string `json:"url"`
	Rev       string `json:"rev"`
	InputRev  string `json:"inputRev"`
	Type      string `json:"type"`
	Inherited bool   `json:"inherited"`
}

type lakeManifest struct {
	Packages []lakeManifestPackage `json:"packages"`
}

func (p *lakeManifestParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var manifest lakeManifest
	if err := json.Unmarshal(content, &manifest); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, pkg := range manifest.Packages {
		if pkg.Type == "path" {
			continue
		}

		name := strings.Trim(pkg.Name, "«»")
		version := pkg.Rev
		if version == "" {
			version = pkg.InputRev
		}

		deps = append(deps, core.Dependency{
			Name:        scopedName(pkg.Scope, name),
			Version:     version,
			Scope:       core.Runtime,
			Direct:      !pkg.Inherited,
			RegistryURL: pkg.URL,
		})
	}

	return deps, nil
}
