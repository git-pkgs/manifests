package docker

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("docker", core.Manifest, &dockerfileParser{}, core.ExactMatch("Dockerfile"))
	core.Register("docker", core.Manifest, &dockerfileParser{}, core.PrefixMatch("Dockerfile."))
	core.Register("docker", core.Manifest, &dockerComposeParser{}, core.ExactMatch("docker-compose.yml"))
	core.Register("docker", core.Manifest, &dockerComposeParser{}, core.ExactMatch("docker-compose.yaml"))
	core.Register("docker", core.Manifest, &dockerComposeParser{}, core.ExactMatch("compose.yml"))
	core.Register("docker", core.Manifest, &dockerComposeParser{}, core.ExactMatch("compose.yaml"))
}

// dockerfileParser parses Dockerfile files.
type dockerfileParser struct{}

var (
	// FROM image:tag or FROM image:tag AS name or FROM image@digest
	dockerFromRegex = regexp.MustCompile(`(?i)^FROM\s+(\S+)`)
)

func (p *dockerfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		if match := dockerFromRegex.FindStringSubmatch(line); match != nil {
			image := match[1]
			// Skip ARG references like ${BASE_IMAGE}
			if strings.Contains(image, "$") {
				continue
			}

			name, version := core.ParseDockerImage(image)
			if name != "" {
				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
					Scope:   core.Runtime,
					Direct:  true,
				})
			}
		}
	}

	return deps, nil
}

// dockerComposeParser parses docker-compose.yml files.
type dockerComposeParser struct{}

type dockerCompose struct {
	Services map[string]dockerComposeService `yaml:"services"`
}

type dockerComposeService struct {
	Image string `yaml:"image"`
	Build any    `yaml:"build"`
}

func (p *dockerComposeParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var compose dockerCompose
	if err := yaml.Unmarshal(content, &compose); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, service := range compose.Services {
		if service.Image == "" {
			continue
		}

		// Skip ARG references
		if strings.Contains(service.Image, "$") {
			continue
		}

		name, version := core.ParseDockerImage(service.Image)
		if name == "" {
			continue
		}

		// Deduplicate
		key := name + "@" + version
		if seen[key] {
			continue
		}
		seen[key] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}

