package carthage

import (
	"os"
	"testing"
)

func TestCartfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/carthage/Cartfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cartfileParser{}
	deps, err := parser.Parse("Cartfile", content)
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

func TestCartfileResolved(t *testing.T) {
	content, err := os.ReadFile("../../testdata/carthage/Cartfile.resolved")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cartfileResolvedParser{}
	deps, err := parser.Parse("Cartfile.resolved", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check for version information
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
