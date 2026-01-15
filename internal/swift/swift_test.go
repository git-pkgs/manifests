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

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 3 packages (extracted from git URLs)
	expected := []string{
		"vapor",
		"Tasks",
		"Environment",
	}

	for _, name := range expected {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
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

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check Yams
	if dep, ok := depMap["Yams"]; !ok {
		t.Error("expected Yams dependency")
	} else {
		if dep.Version != "5.0.1" {
			t.Errorf("Yams version = %q, want %q", dep.Version, "5.0.1")
		}
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

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 2 packages with versions
	expected := map[string]string{
		"cryptoswift":       "1.6.0",
		"swift-docc-plugin": "1.0.0",
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
