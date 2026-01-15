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

	if len(deps) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 7 packages with versions (docker://node appears twice, map keeps last)
	expected := map[string]string{
		"docker://replicated/dockerfilelint": "latest",
		"actions/docker/cli":                 "master",
		"docker://redis":                     "5",
		"docker://postgres":                  "10",
		"actions/bin/shellcheck":             "master",
	}

	for name, wantVer := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != wantVer {
			t.Errorf("%s version = %q, want %q", name, dep.Version, wantVer)
		}
	}

	// docker://node appears twice with different versions (digest and tag)
	// map will only have one entry, verify it exists with some version
	if dep, ok := depMap["docker://node"]; !ok {
		t.Error("expected docker://node dependency")
	} else if dep.Version == "" {
		t.Error("expected docker://node to have version")
	}
}
