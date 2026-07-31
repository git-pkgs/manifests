package github_actions

import (
	"sort"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("github-actions", core.Lockfile, &actionsLockParser{}, core.ExactMatch("actions.lock"))
}

// actionsLockParser parses the GitHub Actions dependency lockfile written by
// gh actions-lock to .github/workflows/actions.lock. The format is pre-1.0;
// this parser is deliberately lenient (unknown fields ignored, no version
// gate) so older git-pkgs binaries keep working across schema bumps.
//
// See https://github.com/github/actions-lockfile for the schema.
type actionsLockParser struct{}

type actionsLockFile struct {
	Version      string                       `yaml:"version"`
	Workflows    map[string][]string          `yaml:"workflows"`
	Dependencies map[string]actionsLockAction `yaml:"dependencies"`
}

type actionsLockAction struct {
	Ref    string `yaml:"ref"`
	Commit string `yaml:"commit"`
}

func (p *actionsLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock actionsLockFile
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	// Pin keys that appear in a workflows: list are direct dependencies of
	// the repo; the rest arrived transitively via an action's uses: list.
	direct := make(map[string]bool)
	for _, pins := range lock.Workflows {
		for _, pin := range pins {
			direct[pin] = true
		}
	}

	// Sort keys so output order is stable across runs.
	keys := make([]string, 0, len(lock.Dependencies))
	for k := range lock.Dependencies {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	deps := make([]core.Dependency, 0, len(keys))
	for _, key := range keys {
		action := lock.Dependencies[key]
		name, keyRef := splitPin(key)
		if name == "" {
			continue
		}
		// Prefer the entry's ref: field (the resolved tag). Fall back to the
		// ref segment of the pin key when it's absent.
		version := action.Ref
		if version == "" {
			version = keyRef
		}
		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:     core.Runtime,
			Integrity: action.Commit,
			Direct:    direct[key],
		})
	}

	return &core.Result{Dependencies: deps}, nil
}

// splitPin splits an OWNER/REPO@REF pin key into name and ref.
func splitPin(key string) (name, ref string) {
	if idx := strings.Index(key, "@"); idx > 0 {
		return key[:idx], key[idx+1:]
	}
	return key, ""
}
