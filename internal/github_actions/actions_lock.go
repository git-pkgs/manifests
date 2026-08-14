package github_actions

import (
	"sort"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("github-actions", core.Lockfile, &actionsLockParser{}, actionsLockMatch)
}

const actionsLockPath = ".github/workflows/actions.lock"

func actionsLockMatch(filename string) bool {
	path := strings.ReplaceAll(filename, `\`, "/")
	return path == actionsLockPath || strings.HasSuffix(path, "/"+actionsLockPath)
}

// actionsLockParser parses the GitHub Actions dependency lockfile written by
// gh actions-lock to .github/workflows/actions.lock. The format is pre-1.0;
// this parser ignores unknown fields for forward compatibility while handling
// known version-specific fields.
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
	Tag    string `yaml:"tag"`
	Branch string `yaml:"branch"`
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
		version := actionsLockVersion(lock.Version, keyRef, action)
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

func actionsLockVersion(lockVersion, keyRef string, action actionsLockAction) string {
	if lockVersion != "v0.0.1" {
		if action.Ref != "" {
			return action.Ref
		}
		return keyRef
	}

	if action.Tag != "" {
		return action.Tag
	}
	if action.Branch != "" {
		return action.Branch
	}
	if action.Commit != "" {
		return strings.TrimSuffix(keyRef, ":"+action.Commit)
	}
	return keyRef
}

// splitPin splits an OWNER/REPO@REF pin key into name and ref.
func splitPin(key string) (name, ref string) {
	if idx := strings.Index(key, "@"); idx > 0 {
		return key[:idx], key[idx+1:]
	}
	return key, ""
}
