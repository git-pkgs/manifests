package swift

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPackageSwift(t *testing.T) {
	content, err := os.ReadFile("../../testdata/swift/Package.swift")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &packageSwiftParser{}
	deps, err := parser.Parse("Package.swift", content)
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

func TestPackageResolved(t *testing.T) {
	content, err := os.ReadFile("../../testdata/swift/Package.resolved")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &packageResolvedParser{}
	deps, err := parser.Parse("Package.resolved", content)
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

func TestPackageResolvedV2(t *testing.T) {
	content, err := os.ReadFile("../../testdata/swift/Package.resolved.2")
	if err != nil {
		t.Skipf("v2 fixture not found: %v", err)
	}

	parser := &packageResolvedParser{}
	deps, err := parser.Parse("Package.resolved", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}
