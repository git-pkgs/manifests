package precommit

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("pre-commit", core.Manifest, &preCommitConfigParser{}, core.ExactMatch(".pre-commit-config.yaml"))
}

type preCommitConfigParser struct{}

type preCommitConfig struct {
	Repos []preCommitRepo `yaml:"repos"`
}

type preCommitRepo struct {
	Repo string `yaml:"repo"`
	Rev  string `yaml:"rev"`
}

func (p *preCommitConfigParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var config preCommitConfig
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, repo := range config.Repos {
		if repo.Repo == "local" || repo.Repo == "meta" {
			continue
		}

		name := repo.Repo
		if i := strings.Index(name, "://"); i >= 0 {
			name = name[i+3:]
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: repo.Rev,
			Scope:   core.Development,
			Direct:  true,
		})
	}

	return deps, nil
}
