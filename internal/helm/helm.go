package helm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("helm", core.Manifest, &chartParser{}, core.ExactMatch("Chart.yaml"))
	core.Register("helm", core.Lockfile, &chartLockParser{}, core.ExactMatch("Chart.lock"))
}

type chartParser struct{}

type chartMetadata struct {
	Name         string            `yaml:"name"`
	Version      string            `yaml:"version"`
	Dependencies []chartDependency `yaml:"dependencies"`
}

type chartDependency struct {
	Name       string `yaml:"name"`
	Version    string `yaml:"version"`
	Repository string `yaml:"repository"`
}

func (p *chartParser) Parse(filename string, content []byte) (*core.Result, error) {
	var chart chartMetadata
	if err := yaml.Unmarshal(content, &chart); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	return &core.Result{
		Name:         chart.Name,
		Version:      chart.Version,
		Dependencies: helmDependencies(chart.Dependencies),
	}, nil
}

type chartLockParser struct{}

type chartLock struct {
	Dependencies []chartDependency `yaml:"dependencies"`
	Digest       string            `yaml:"digest"`
}

func (p *chartLockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock chartLock
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	return &core.Result{
		Digest:       lock.Digest,
		Dependencies: helmDependencies(lock.Dependencies),
	}, nil
}

func helmDependencies(entries []chartDependency) []core.Dependency {
	dependencies := make([]core.Dependency, 0, len(entries))
	for _, entry := range entries {
		if entry.Name == "" {
			continue
		}
		dependencies = append(dependencies, core.Dependency{
			Name:        entry.Name,
			Version:     entry.Version,
			Scope:       core.Runtime,
			Direct:      true,
			RegistryURL: entry.Repository,
		})
	}
	return dependencies
}
