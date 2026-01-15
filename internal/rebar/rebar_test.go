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

	// All 3 packages with versions
	expected := map[string]string{
		"hex_core":       "0.10.3",
		"verl":           "1.1.1",
		"ssl_verify_fun": "1.1.7",
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
}
