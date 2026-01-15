package crystal

import (
	"github.com/git-pkgs/manifests/internal/core"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("crystal", core.Manifest, &shardYMLParser{}, core.ExactMatch("shard.yml"))
	core.Register("crystal", core.Lockfile, &shardLockParser{}, core.ExactMatch("shard.lock"))
}

// extractShardName extracts shard name from "  name:" line
func extractShardName(line string) (string, bool) {
	if len(line) < 4 || line[0] != ' ' || line[1] != ' ' || line[2] == ' ' {
		return "", false
	}
	if line[len(line)-1] != ':' {
		return "", false
	}
	return line[2 : len(line)-1], true
}

// extractShardValue extracts value from "    key: value" lines
func extractShardValue(line, prefix string) (string, bool) {
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	return line[len(prefix):], true
}

// shardYMLParser parses shard.yml files.
type shardYMLParser struct{}

type shardYML struct {
	Dependencies            map[string]shardDep `yaml:"dependencies"`
	DevelopmentDependencies map[string]shardDep `yaml:"development_dependencies"`
}

type shardDep struct {
	GitHub  string `yaml:"github"`
	GitLab  string `yaml:"gitlab"`
	Git     string `yaml:"git"`
	Path    string `yaml:"path"`
	Version string `yaml:"version"`
	Branch  string `yaml:"branch"`
	Tag     string `yaml:"tag"`
	Commit  string `yaml:"commit"`
}

func (p *shardYMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var shard shardYML
	if err := yaml.Unmarshal(content, &shard); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for name, dep := range shard.Dependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: getShardVersion(dep),
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	for name, dep := range shard.DevelopmentDependencies {
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: getShardVersion(dep),
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return deps, nil
}

func getShardVersion(dep shardDep) string {
	if dep.Version != "" {
		return dep.Version
	}
	if dep.Tag != "" {
		return dep.Tag
	}
	if dep.Branch != "" {
		return dep.Branch
	}
	return ""
}

// shardLockParser parses shard.lock files using regex for speed.
type shardLockParser struct{}

func (p *shardLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))

	inShards := false
	var currentName string
	var currentVersion string

	core.ForEachLine(text, func(line string) bool {
		// Detect shards: section
		if line == "shards:" {
			inShards = true
			return true
		}

		if !inShards {
			return true
		}

		// Shard name (2-space indent)
		if name, ok := extractShardName(line); ok {
			// Save previous shard if any
			if currentName != "" {
				deps = append(deps, core.Dependency{
					Name:    currentName,
					Version: currentVersion,
					Scope:   core.Runtime,
					Direct:  false,
				})
			}
			currentName = name
			currentVersion = ""
			return true
		}

		// Version or commit (4-space indent)
		if currentName != "" {
			if v, ok := extractShardValue(line, "    version: "); ok {
				currentVersion = v
			} else if v, ok := extractShardValue(line, "    commit: "); ok {
				if currentVersion == "" { // version takes precedence
					currentVersion = v
				}
			}
		}
		return true
	})

	// Don't forget the last shard
	if currentName != "" {
		deps = append(deps, core.Dependency{
			Name:    currentName,
			Version: currentVersion,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}
