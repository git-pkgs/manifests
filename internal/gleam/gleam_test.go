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

	// Check runtime dependency
	if dep, ok := depMap["gleam_stdlib"]; !ok {
		t.Error("expected gleam_stdlib dependency")
	} else {
		if dep.Version != ">= 0.53.0 and < 2.0.0" {
			t.Errorf("expected gleam_stdlib version >= 0.53.0 and < 2.0.0, got %s", dep.Version)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("expected gleam_stdlib scope Runtime, got %v", dep.Scope)
		}
	}

	// Check dev dependency
	if dep, ok := depMap["gleeunit"]; !ok {
		t.Error("expected gleeunit dependency")
	} else {
		if dep.Scope != core.Development {
			t.Errorf("expected gleeunit scope Development, got %v", dep.Scope)
		}
	}
}
