package brew

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestBrewfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/brew/Brewfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &brewfileParser{}
	deps, err := parser.Parse("Brewfile", content)
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

	// Check brew formula
	if shellcheck, ok := depMap["shellcheck"]; !ok {
		t.Error("expected shellcheck dependency")
	} else {
		if shellcheck.Scope != core.Runtime {
			t.Errorf("shellcheck scope = %q, want %q", shellcheck.Scope, core.Runtime)
		}
	}

	// Check tap
	if tap, ok := depMap["peripheryapp/periphery"]; !ok {
		t.Error("expected peripheryapp/periphery tap")
	} else {
		if tap.Scope != core.Runtime {
			t.Errorf("tap scope = %q, want %q", tap.Scope, core.Runtime)
		}
	}

	// Check cask
	if periphery, ok := depMap["periphery"]; !ok {
		t.Error("expected periphery cask")
	} else {
		if periphery.Scope != core.Runtime {
			t.Errorf("periphery scope = %q, want %q", periphery.Scope, core.Runtime)
		}
	}

	// Verify comment was filtered out
	if _, ok := depMap["swiftlint"]; ok {
		t.Error("commented swiftlint should not be included")
	}
}

func TestBrewfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/brew/Brewfile.lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &brewfileLockParser{}
	deps, err := parser.Parse("Brewfile.lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check that we have version information
	hasVersion := false
	for _, d := range deps {
		if d.Version != "" {
			hasVersion = true
			break
		}
	}

	if !hasVersion {
		t.Error("expected at least one dependency with version")
	}
}
