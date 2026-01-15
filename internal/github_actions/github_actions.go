package github_actions

import (
	"github.com/git-pkgs/manifests/internal/core"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func init() {
	core.Register("github-actions", core.Manifest, &githubWorkflowParser{}, githubWorkflowMatch)
}

// githubWorkflowMatch matches GitHub workflow files in .github/workflows/
func githubWorkflowMatch(filename string) bool {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)

	// Check extension first
	if ext != ".yml" && ext != ".yaml" {
		return false
	}

	// Check if in .github/workflows directory
	dir := filepath.Dir(filename)
	if strings.HasSuffix(dir, ".github/workflows") ||
		strings.HasSuffix(dir, ".github\\workflows") ||
		dir == ".github/workflows" ||
		dir == ".github\\workflows" {
		return true
	}

	// Also match just the filename for testing
	return base == "workflow.yml" || base == "workflow.yaml"
}

// githubWorkflowParser parses GitHub Actions workflow files.
type githubWorkflowParser struct{}

type githubWorkflow struct {
	Jobs map[string]githubJob `yaml:"jobs"`
}

type githubJob struct {
	Container any            `yaml:"container"`
	Services  map[string]any `yaml:"services"`
	Steps     []githubStep   `yaml:"steps"`
}

type githubStep struct {
	Uses string `yaml:"uses"`
}

func (p *githubWorkflowParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var workflow githubWorkflow
	if err := yaml.Unmarshal(content, &workflow); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	seen := make(map[string]bool)

	for _, job := range workflow.Jobs {
		// Parse step actions
		for _, step := range job.Steps {
			if step.Uses == "" {
				continue
			}

			name, version := parseGitHubAction(step.Uses)
			if name == "" {
				continue
			}

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

		// Parse job container
		if job.Container != nil {
			var containerImage string
			switch c := job.Container.(type) {
			case string:
				containerImage = c
			case map[string]any:
				if img, ok := c["image"].(string); ok {
					containerImage = img
				}
			}

			if containerImage != "" && !strings.Contains(containerImage, "$") {
				name, version := core.ParseDockerImage(containerImage)
				if name != "" {
					key := "docker://" + name + "@" + version
					if !seen[key] {
						seen[key] = true
						deps = append(deps, core.Dependency{
							Name:    "docker://" + name,
							Version: version,
							Scope:   core.Runtime,
							Direct:  true,
						})
					}
				}
			}
		}

		// Parse services
		for _, service := range job.Services {
			var serviceImage string
			switch s := service.(type) {
			case string:
				serviceImage = s
			case map[string]any:
				if img, ok := s["image"].(string); ok {
					serviceImage = img
				}
			}

			if serviceImage == "" || strings.Contains(serviceImage, "$") {
				continue
			}

			name, version := core.ParseDockerImage(serviceImage)
			if name == "" {
				continue
			}

			key := "docker://" + name + "@" + version
			if seen[key] {
				continue
			}
			seen[key] = true

			deps = append(deps, core.Dependency{
				Name:    "docker://" + name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// parseGitHubAction parses a GitHub Action reference.
// Formats: owner/repo@ref, owner/repo/path@ref, docker://image:tag
func parseGitHubAction(uses string) (name, version string) {
	// Skip docker:// references (handled separately)
	if strings.HasPrefix(uses, "docker://") {
		image := strings.TrimPrefix(uses, "docker://")
		imgName, imgVersion := core.ParseDockerImage(image)
		return "docker://" + imgName, imgVersion
	}

	// Skip local actions
	if strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "../") {
		return "", ""
	}

	// Parse owner/repo@ref or owner/repo/path@ref
	if idx := strings.Index(uses, "@"); idx > 0 {
		return uses[:idx], uses[idx+1:]
	}

	return uses, ""
}
