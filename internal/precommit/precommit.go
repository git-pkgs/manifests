package precommit

import (
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("pre-commit", core.Manifest, &preCommitYAMLParser{}, core.ExactMatch(".pre-commit-config.yaml"))
	core.Register("pre-commit", core.Manifest, &prekTOMLParser{}, core.ExactMatch("prek.toml"))
}

type repo struct {
	Repo string
	Rev  string
}

func reposToDeps(repos []repo) []core.Dependency {
	var deps []core.Dependency
	for _, r := range repos {
		if r.Repo == "local" || r.Repo == "meta" || r.Repo == "builtin" {
			continue
		}

		name := r.Repo
		if i := strings.Index(name, "://"); i >= 0 {
			name = name[i+3:]
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: r.Rev,
			Scope:   core.Development,
			Direct:  true,
		})
	}
	return deps
}

// YAML parser for .pre-commit-config.yaml

type preCommitYAMLParser struct{}

type preCommitYAMLConfig struct {
	Repos []repo `yaml:"repos"`
}

func (p *preCommitYAMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var config preCommitYAMLConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
	return reposToDeps(config.Repos), nil
}

// TOML parser for prek.toml

type prekTOMLParser struct{}

type prekTOMLConfig struct {
	Repos []repo `toml:"repos"`
}

func (p *prekTOMLParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var config prekTOMLConfig
	if err := toml.Unmarshal(content, &config); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
	return reposToDeps(config.Repos), nil
}
