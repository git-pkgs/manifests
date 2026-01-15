package cocoapods

import (
	"os"
	"testing"
)

func TestPodfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cocoapods/Podfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &podfileParser{}
	deps, err := parser.Parse("Podfile", content)
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

func TestPodfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cocoapods/Podfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &podfileLockParser{}
	deps, err := parser.Parse("Podfile.lock", content)
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

func TestPodspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cocoapods/example.podspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &podspecParser{}
	_, err = parser.Parse("example.podspec", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Podspec may or may not have dependencies
	// Just check it parses without error
}
