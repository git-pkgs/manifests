package nimble

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestNimble(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nimble/example.nimble")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &nimbleParser{}
	deps, err := parser.Parse("example.nimble", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check nim with >= constraint
	if dep, ok := depMap["nim"]; !ok {
		t.Error("expected nim dependency")
	} else if dep.Version != ">= 1.6.0" {
		t.Errorf("expected nim version >= 1.6.0, got %s", dep.Version)
	}

	// Check chronos
	if dep, ok := depMap["chronos"]; !ok {
		t.Error("expected chronos dependency")
	} else if dep.Version != ">= 3.0.0" {
		t.Errorf("expected chronos version >= 3.0.0, got %s", dep.Version)
	}

	// Check multiple deps on one line
	if dep, ok := depMap["chronicles"]; !ok {
		t.Error("expected chronicles dependency")
	} else if dep.Version != ">= 0.10.0" {
		t.Errorf("expected chronicles version >= 0.10.0, got %s", dep.Version)
	}

	if dep, ok := depMap["stew"]; !ok {
		t.Error("expected stew dependency")
	} else if dep.Version != ">= 0.1.0" {
		t.Errorf("expected stew version >= 0.1.0, got %s", dep.Version)
	}

	// Check results without version
	if dep, ok := depMap["results"]; !ok {
		t.Error("expected results dependency")
	} else if dep.Version != "" {
		t.Errorf("expected results version empty, got %s", dep.Version)
	}
}
