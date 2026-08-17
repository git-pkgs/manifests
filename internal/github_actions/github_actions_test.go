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
	res, err := parser.Parse("workflow.yml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
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

func TestGitHubWorkflowDeclarations(t *testing.T) {
	content := []byte(`jobs:
  build:
    steps:
      - uses: actions/checkout@v4
      - uses: actions/cache/restore@v3
      - uses: actions/checkout@v3
      - uses: docker://alpine:3.20
      - uses: ./local-action
  test:
    steps:
      - uses: actions/checkout@main
`)

	result, err := (&githubWorkflowParser{}).Parse("workflow.yml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := map[string]struct {
		name    string
		version string
	}{
		"jobs/build/steps/actions%2Fcheckout":        {"actions/checkout", "v4"},
		"jobs/build/steps/actions%2Fcache%2Frestore": {"actions/cache/restore", "v3"},
		"jobs/build/steps/actions%2Fcheckout/2":      {"actions/checkout", "v3"},
		"jobs/test/steps/actions%2Fcheckout":         {"actions/checkout", "main"},
	}
	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		expected, ok := want[declaration.Location]
		if !ok {
			t.Errorf("unexpected declaration at %q: %+v", declaration.Location, declaration)
			continue
		}
		if declaration.Name != expected.name || declaration.Version != expected.version || declaration.Scope != core.Runtime {
			t.Errorf("declaration at %q = %+v, want name %q and version %q",
				declaration.Location, declaration, expected.name, expected.version)
		}
	}
}
