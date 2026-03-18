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
		deps = collectStepActions(job.Steps, deps, seen)
		deps = collectContainerImage(job.Container, deps, seen)
		deps = collectServiceImages(job.Services, deps, seen)
	}

	return deps, nil
}

// collectStepActions extracts action dependencies from job steps.
func collectStepActions(steps []githubStep, deps []core.Dependency, seen map[string]bool) []core.Dependency {
	for _, step := range steps {
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
	return deps
}

// collectContainerImage extracts a Docker dependency from a job's container field.
func collectContainerImage(container any, deps []core.Dependency, seen map[string]bool) []core.Dependency {
	if container == nil {
		return deps
	}

	image := extractImageRef(container)
	if image == "" || strings.Contains(image, "$") {
		return deps
	}

	return appendDockerDep(image, deps, seen)
}

// collectServiceImages extracts Docker dependencies from a job's services.
func collectServiceImages(services map[string]any, deps []core.Dependency, seen map[string]bool) []core.Dependency {
	for _, service := range services {
		image := extractImageRef(service)
		if image == "" || strings.Contains(image, "$") {
			continue
		}
		deps = appendDockerDep(image, deps, seen)
	}
	return deps
}

// extractImageRef pulls a Docker image string from either a plain string or a map with an "image" key.
func extractImageRef(v any) string {
	switch c := v.(type) {
	case string:
		return c
	case map[string]any:
		if img, ok := c["image"].(string); ok {
			return img
		}
	}
	return ""
}

// appendDockerDep parses a Docker image reference and appends it as a dependency if not already seen.
func appendDockerDep(image string, deps []core.Dependency, seen map[string]bool) []core.Dependency {
	name, version := core.ParseDockerImage(image)
	if name == "" {
		return deps
	}

	key := "docker://" + name + "@" + version
	if seen[key] {
		return deps
	}
	seen[key] = true

	return append(deps, core.Dependency{
		Name:    "docker://" + name,
		Version: version,
		Scope:   core.Runtime,
		Direct:  true,
	})
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
