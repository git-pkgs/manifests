package hex

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestMixExs(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hex/mix.exs")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &mixExsParser{}
	deps, err := parser.Parse("mix.exs", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check that dependencies have names
	for _, d := range deps {
		if d.Name == "" {
			t.Error("expected all dependencies to have names")
		}
	}
}

func TestMixLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hex/mix.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &mixLockParser{}
	deps, err := parser.Parse("mix.lock", content)
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

	// Check for version and integrity
	hasVersion := false
	hasIntegrity := false
	for _, d := range deps {
		if d.Version != "" {
			hasVersion = true
		}
		if d.Integrity != "" {
			hasIntegrity = true
		}
	}
	if !hasVersion {
		t.Error("expected at least one dependency with version")
	}
	if !hasIntegrity {
		t.Error("expected at least one dependency with integrity hash")
	}
}
