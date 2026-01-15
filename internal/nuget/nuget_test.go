package nuget

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestCsproj(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example.csproj")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &csprojParser{}
	deps, err := parser.Parse("example.csproj", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 8 {
		t.Fatalf("expected 8 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check PackageReference with Version attribute
	if dep, ok := depMap["Microsoft.AspNetCore"]; !ok {
		t.Error("expected Microsoft.AspNetCore dependency")
	} else if dep.Version != "1.1.1" {
		t.Errorf("expected version 1.1.1, got %s", dep.Version)
	}

	// Check PackageReference with Version element
	if dep, ok := depMap["System.Resources.Extensions"]; !ok {
		t.Error("expected System.Resources.Extensions dependency")
	} else if dep.Version != "4.7.0" {
		t.Errorf("expected version 4.7.0, got %s", dep.Version)
	}
}

func TestNuspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example.nuspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &nuspecParser{}
	deps, err := parser.Parse("example.nuspec", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check dependency with version
	if dep, ok := depMap["FubuCore"]; !ok {
		t.Error("expected FubuCore dependency")
	} else if dep.Version != "3.2.0.3001" {
		t.Errorf("expected version 3.2.0.3001, got %s", dep.Version)
	}

	// Check dependency without version
	if dep, ok := depMap["DotNetZip"]; !ok {
		t.Error("expected DotNetZip dependency")
	} else if dep.Version != "" {
		t.Errorf("expected version empty, got %s", dep.Version)
	}
}

func TestPackagesConfig(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/packages.config")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &packagesConfigParser{}
	deps, err := parser.Parse("packages.config", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check regular dependency
	if dep, ok := depMap["AutoMapper"]; !ok {
		t.Error("expected AutoMapper dependency")
	} else if dep.Version != "2.1.267" {
		t.Errorf("expected version 2.1.267, got %s", dep.Version)
	}

	// Check development dependency
	if dep, ok := depMap["Microsoft.Net.Compilers"]; !ok {
		t.Error("expected Microsoft.Net.Compilers dependency")
	} else if dep.Scope != core.Development {
		t.Errorf("expected scope Development, got %v", dep.Scope)
	}
}

func TestProjectAssets(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/nuget_project.assets.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &projectAssetsParser{}
	deps, err := parser.Parse("project.assets.json", content)
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

	// Check package a
	if dep, ok := depMap["a"]; !ok {
		t.Error("expected a dependency")
	} else if dep.Version != "1.0.0" {
		t.Errorf("a version = %q, want %q", dep.Version, "1.0.0")
	}

	// Check package b
	if dep, ok := depMap["b"]; !ok {
		t.Error("expected b dependency")
	} else if dep.Version != "1.0.0" {
		t.Errorf("b version = %q, want %q", dep.Version, "1.0.0")
	}

	// Check package c from net2.2 framework
	if dep, ok := depMap["c"]; !ok {
		t.Error("expected c dependency")
	} else if dep.Version != "1.0.0" {
		t.Errorf("c version = %q, want %q", dep.Version, "1.0.0")
	}

	// Verify project reference is excluded
	if _, ok := depMap["another_project"]; ok {
		t.Error("project reference should be excluded")
	}
}

func TestPaketLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/paket.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &paketLockParser{}
	deps, err := parser.Parse("paket.lock", content)
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

	// Check Argu
	if dep, ok := depMap["Argu"]; !ok {
		t.Error("expected Argu dependency")
	} else if dep.Version != "2.1" {
		t.Errorf("expected version 2.1, got %s", dep.Version)
	}

	// Check Newtonsoft.Json
	if dep, ok := depMap["Newtonsoft.Json"]; !ok {
		t.Error("expected Newtonsoft.Json dependency")
	} else if dep.Version != "9.0.1" {
		t.Errorf("expected version 9.0.1, got %s", dep.Version)
	}
}
