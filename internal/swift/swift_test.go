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
	res, err := parser.Parse("Package.swift", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	expected := []string{
		"github.com/qutheory/vapor",
		"github.com/czechboy0/Tasks",
		"github.com/czechboy0/Environment",
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
	res, err := parser.Parse("Package.resolved", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	if dep, ok := depMap["github.com/jpsim/Yams"]; !ok {
		t.Error("expected github.com/jpsim/Yams dependency")
	} else if dep.Version != "5.0.1" {
		t.Errorf("Yams version = %q, want %q", dep.Version, "5.0.1")
	}
}

func TestPackageResolvedV2(t *testing.T) {
	content, err := os.ReadFile("../../testdata/swift/Package.resolved.2")
	if err != nil {
		t.Skipf("v2 fixture not found: %v", err)
	}

	parser := &packageResolvedParser{}
	res, err := parser.Parse("Package.resolved", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	expected := map[string]string{
		"github.com/krzyzanowskim/CryptoSwift": "1.6.0",
		"github.com/apple/swift-docc-plugin":   "1.0.0",
		"apple.swift-argument-parser":          "1.2.3",
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

func TestSwiftSourceCoordinate(t *testing.T) {
	tests := map[string]string{
		"https://github.com/apple/swift-argument-parser.git": "github.com/apple/swift-argument-parser",
		"ssh://git@github.com/apple/swift-nio.git":           "github.com/apple/swift-nio",
		"git@github.com:apple/swift-log.git":                 "github.com/apple/swift-log",
		"../local-package":                                   "",
	}

	for rawURL, want := range tests {
		if got := swiftSourceCoordinate(rawURL); got != want {
			t.Errorf("swiftSourceCoordinate(%q) = %q, want %q", rawURL, got, want)
		}
	}
}
