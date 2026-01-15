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

	if len(deps) != 6 {
		t.Fatalf("expected 6 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 6 packages
	expected := []string{
		"markdownlint-cli",
		"shellcheck",
		"shfmt",
		"swiftformat",
		"peripheryapp/periphery", // tap
		"periphery",              // cask
	}

	for _, name := range expected {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
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

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 5 packages with versions
	expected := map[string]string{
		"markdownlint-cli": "0.32.2",
		"shellcheck":       "0.8.0",
		"shfmt":            "3.5.1",
		"swiftformat":      "0.49.18",
		"periphery":        "2.9.0",
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
