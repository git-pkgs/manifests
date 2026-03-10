package precommit

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPreCommitConfig(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pre-commit/.pre-commit-config.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &preCommitYAMLParser{}
	deps, err := parser.Parse(".pre-commit-config.yaml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	expected := map[string]string{
		"github.com/pre-commit/pre-commit-hooks": "v4.5.0",
		"github.com/psf/black":                   "24.2.0",
		"github.com/PyCQA/flake8":                "7.0.0",
		"gitlab.com/pycqa/flake8":                "5.0.0",
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
		if dep.Scope != core.Development {
			t.Errorf("%s scope = %q, want %q", name, dep.Scope, core.Development)
		}
		if !dep.Direct {
			t.Errorf("%s should be direct", name)
		}
	}
}

func TestPrekTOML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pre-commit/prek.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &prekTOMLParser{}
	deps, err := parser.Parse("prek.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	expected := map[string]string{
		"github.com/crate-ci/typos":          "v1.42.3",
		"github.com/executablebooks/mdformat": "1.0.0",
		"gitlab.com/pycqa/flake8":            "7.0.0",
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
		if dep.Scope != core.Development {
			t.Errorf("%s scope = %q, want %q", name, dep.Scope, core.Development)
		}
		if !dep.Direct {
			t.Errorf("%s should be direct", name)
		}
	}
}
