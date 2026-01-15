package nix

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"
)

func init() {
	core.Register("nix", core.Manifest, &flakeNixParser{}, core.ExactMatch("flake.nix"))
	core.Register("nix", core.Lockfile, &flakeLockParser{}, core.ExactMatch("flake.lock"))
	core.Register("nix", core.Lockfile, &sourcesJSONParser{}, core.ExactMatch("sources.json"))
}

// flakeNixParser parses flake.nix files.
type flakeNixParser struct{}

var (
	// Match: name.url = "github:owner/repo" or name = { url = "github:owner/repo"; }
	// Note: names can contain hyphens like "flake-utils"
	flakeInputURLRegex = regexp.MustCompile(`([\w-]+)\.url\s*=\s*"([^"]+)"`)
	flakeInputRegex    = regexp.MustCompile(`([\w-]+)\s*=\s*\{\s*url\s*=\s*"([^"]+)"`)
)

func (p *flakeNixParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	// Find inputs section
	inputsStart := strings.Index(text, "inputs")
	if inputsStart == -1 {
		return nil, nil
	}

	// Parse name.url = "..." pattern
	for _, match := range flakeInputURLRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		url := match[2]

		if seen[name] {
			continue
		}
		seen[name] = true

		version := parseFlakeURL(url)

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	// Parse name = { url = "..."; } pattern
	for _, match := range flakeInputRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		url := match[2]

		if seen[name] {
			continue
		}
		seen[name] = true

		version := parseFlakeURL(url)

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}

// parseFlakeURL extracts version info from a flake URL
func parseFlakeURL(url string) string {
	// Format: github:owner/repo/ref or github:owner/repo
	if strings.HasPrefix(url, "github:") {
		parts := strings.Split(strings.TrimPrefix(url, "github:"), "/")
		if len(parts) >= 3 {
			return parts[2]
		}
		return url
	}
	return url
}

// flakeLockParser parses flake.lock files.
type flakeLockParser struct{}

type flakeLock struct {
	Nodes map[string]flakeLockNode `json:"nodes"`
	Root  string                   `json:"root"`
}

type flakeLockNode struct {
	Locked struct {
		Owner       string `json:"owner"`
		Repo        string `json:"repo"`
		Rev         string `json:"rev"`
		Type        string `json:"type"`
		NarHash     string `json:"narHash"`
		Ref         string `json:"ref"`
		LastModifed int64  `json:"lastModified"`
	} `json:"locked"`
	Original struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
		Type  string `json:"type"`
		Ref   string `json:"ref"`
	} `json:"original"`
}

func (p *flakeLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock flakeLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, node := range lock.Nodes {
		// Skip root node
		if name == "root" {
			continue
		}

		// Build dependency name
		depName := name
		if node.Original.Owner != "" && node.Original.Repo != "" {
			depName = node.Original.Owner + "/" + node.Original.Repo
		}

		// Use rev as version, or ref if no rev
		version := node.Locked.Rev
		if version == "" {
			version = node.Locked.Ref
		}

		deps = append(deps, core.Dependency{
			Name:    depName,
			Version: version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

// sourcesJSONParser parses niv sources.json files.
type sourcesJSONParser struct{}

type sourcesSource struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Rev    string `json:"rev"`
	Branch string `json:"branch"`
}

func (p *sourcesJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var sources map[string]sourcesSource
	if err := json.Unmarshal(content, &sources); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, source := range sources {
		// Build dependency name from owner/repo
		depName := name
		if source.Owner != "" && source.Repo != "" {
			depName = source.Owner + "/" + source.Repo
		}

		deps = append(deps, core.Dependency{
			Name:    depName,
			Version: source.Rev,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}
