package gleam

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGleamToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gleam/gleam.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gleamTomlParser{}
	deps, err := parser.Parse("gleam.toml", content)
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

	// All 3 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"gleam_stdlib", ">= 0.53.0 and < 2.0.0", core.Runtime},
		{"gleam_http", "~> 3.0", core.Runtime},
		{"gleeunit", ">= 1.3.0 and < 2.0.0", core.Development},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}
