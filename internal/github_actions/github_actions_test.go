package github_actions

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGitHubWorkflow(t *testing.T) {
	content, err := os.ReadFile("../../testdata/github-actions/workflow.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &githubWorkflowParser{}
	deps, err := parser.Parse("workflow.yml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check GitHub action with @ref
	if dep, ok := depMap["actions/bin/shellcheck"]; !ok {
		t.Error("expected actions/bin/shellcheck dependency")
	} else if dep.Version != "master" {
		t.Errorf("expected version master, got %s", dep.Version)
	}

	// Check docker:// action
	if dep, ok := depMap["docker://replicated/dockerfilelint"]; !ok {
		t.Error("expected docker://replicated/dockerfilelint dependency")
	} else if dep.Version != "latest" {
		t.Errorf("expected version latest, got %s", dep.Version)
	}

	// Check docker node (version varies due to map iteration order - could be digest or tag)
	if dep, ok := depMap["docker://node"]; !ok {
		t.Error("expected docker://node dependency")
	} else if dep.Version == "" {
		t.Error("expected docker://node to have version")
	}

	// Check services
	if dep, ok := depMap["docker://redis"]; !ok {
		t.Error("expected docker://redis service dependency")
	} else if dep.Version != "5" {
		t.Errorf("expected redis version 5, got %s", dep.Version)
	}

	if dep, ok := depMap["docker://postgres"]; !ok {
		t.Error("expected docker://postgres service dependency")
	} else if dep.Version != "10" {
		t.Errorf("expected postgres version 10, got %s", dep.Version)
	}
}
