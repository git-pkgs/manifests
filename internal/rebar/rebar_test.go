package rebar

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRebarLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/rebar/rebar.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &rebarLockParser{}
	deps, err := parser.Parse("rebar.lock", content)
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

	// Check hex_core
	if dep, ok := depMap["hex_core"]; !ok {
		t.Error("expected hex_core dependency")
	} else if dep.Version != "0.10.3" {
		t.Errorf("expected hex_core version 0.10.3, got %s", dep.Version)
	}

	// Check verl
	if dep, ok := depMap["verl"]; !ok {
		t.Error("expected verl dependency")
	} else if dep.Version != "1.1.1" {
		t.Errorf("expected verl version 1.1.1, got %s", dep.Version)
	}

	// Check ssl_verify_fun
	if dep, ok := depMap["ssl_verify_fun"]; !ok {
		t.Error("expected ssl_verify_fun dependency")
	} else if dep.Version != "1.1.7" {
		t.Errorf("expected ssl_verify_fun version 1.1.7, got %s", dep.Version)
	}
}
