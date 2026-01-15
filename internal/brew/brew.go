package brew

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"
)

func init() {
	core.Register("brew", core.Manifest, &brewfileParser{}, core.ExactMatch("Brewfile"))
	core.Register("brew", core.Lockfile, &brewfileLockParser{}, core.ExactMatch("Brewfile.lock.json"))
}

// brewfileParser parses Brewfile (manifest).
type brewfileParser struct{}

var (
	// brew "name" or brew "name", args
	brewFormulaRegex = regexp.MustCompile(`^\s*brew\s+["']([^"']+)["']`)
	// cask "name"
	brewCaskRegex = regexp.MustCompile(`^\s*cask\s+["']([^"']+)["']`)
	// tap "owner/repo"
	brewTapRegex = regexp.MustCompile(`^\s*tap\s+["']([^"']+)["']`)
)

func (p *brewfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		if match := brewFormulaRegex.FindStringSubmatch(line); match != nil {
			deps = append(deps, core.Dependency{
				Name:   match[1],
				Scope:   core.Runtime,
				Direct: true,
			})
		} else if match := brewCaskRegex.FindStringSubmatch(line); match != nil {
			deps = append(deps, core.Dependency{
				Name:   match[1],
				Scope:   core.Runtime,
				Direct: true,
			})
		} else if match := brewTapRegex.FindStringSubmatch(line); match != nil {
			deps = append(deps, core.Dependency{
				Name:   match[1],
				Scope:   core.Runtime,
				Direct: true,
			})
		}
	}

	return deps, nil
}

// brewfileLockParser parses Brewfile.lock.json.
type brewfileLockParser struct{}

type brewfileLock struct {
	Entries struct {
		Brew map[string]brewLockEntry `json:"brew"`
		Cask map[string]brewLockEntry `json:"cask"`
		Tap  map[string]brewLockEntry `json:"tap"`
	} `json:"entries"`
}

type brewLockEntry struct {
	Version string `json:"version"`
	Bottle  struct {
		Files map[string]struct {
			SHA256 string `json:"sha256"`
		} `json:"files"`
	} `json:"bottle"`
}

func (p *brewfileLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock brewfileLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, entry := range lock.Entries.Brew {
		integrity := ""
		// Get first available SHA256 from bottle files
		for _, file := range entry.Bottle.Files {
			if file.SHA256 != "" {
				integrity = "sha256-" + file.SHA256
				break
			}
		}
		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   entry.Version,
			Scope:   core.Runtime,
			Integrity: integrity,
			Direct:    true,
		})
	}

	for name, entry := range lock.Entries.Cask {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: entry.Version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}
