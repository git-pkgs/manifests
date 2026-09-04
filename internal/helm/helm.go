package helm

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"go.yaml.in/yaml/v3"
)

func init() {
	core.Register("helm", core.Manifest, &chartParser{}, core.ExactMatch("Chart.yaml"))
	core.Register("helm", core.Lockfile, &chartLockParser{}, core.ExactMatch("Chart.lock"))
	core.Register("helm", core.Manifest, &requirementsParser{}, core.ExactMatch("requirements.yaml"))
	core.Register("helm", core.Lockfile, &chartLockParser{}, core.ExactMatch("requirements.lock"))
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

// requirementsParser handles Helm v2 requirements.yaml, which holds only the
// dependencies list that later moved into Chart.yaml.
type requirementsParser struct{}

type requirementsFile struct {
	Dependencies []chartDependency `yaml:"dependencies"`
}

func (p *requirementsParser) Parse(filename string, content []byte) (*core.Result, error) {
	var requirements requirementsFile
	if err := yaml.Unmarshal(content, &requirements); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	return &core.Result{
		Dependencies: helmDependencies(requirements.Dependencies),
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
		registryURL, source := helmRepository(entry.Repository)
		dependencies = append(dependencies, core.Dependency{
			Name:        entry.Name,
			Version:     entry.Version,
			Scope:       core.Runtime,
			Direct:      true,
			RegistryURL: registryURL,
			Source:      source,
		})
	}
	return dependencies
}

// helmRepository classifies a Chart.yaml repository value. file:// paths are
// preserved as Source declarations rather than registry URLs. Everything
// else, including unresolved @name and alias:name references, is kept in
// RegistryURL as declared.
func helmRepository(repo string) (string, core.Source) {
	if strings.HasPrefix(repo, "file://") {
		return "", core.Source{Kind: core.SourcePath, Value: strings.TrimPrefix(repo, "file://")}
	}
	return repo, core.Source{}
}
