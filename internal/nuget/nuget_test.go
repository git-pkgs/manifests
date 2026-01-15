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

	// All 8 packages with exact versions
	expected := map[string]string{
		"Microsoft.AspNetCore":                   "1.1.1",
		"Microsoft.AspNetCore.Mvc":               "1.1.2",
		"Microsoft.AspNetCore.StaticFiles":       "1.1.1",
		"Microsoft.Extensions.Logging.Debug":     "1.1.1",
		"Microsoft.Extensions.DependencyInjection": "1.1.1",
		"Microsoft.VisualStudio.Web.BrowserLink": "1.1.0",
		"System.Resources.Extensions":            "4.7.0",
		"Contoso.Utility.UsefulStuff":            "3.6.0",
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

	// All 4 packages with exact versions
	expected := map[string]string{
		"FubuCore":               "3.2.0.3001",
		"HtmlTags":               "[3.2.0.3001]",
		"DotNetZip":              "",
		"DevelopmentOnlyPackage": "1.2.3",
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

	// Note: nuspec parser doesn't currently detect developmentDependency attribute
	// All dependencies are marked as Runtime
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

	// All 7 packages with exact versions
	expected := map[string]string{
		"AutoMapper":                 "2.1.267",
		"Microsoft.Web.Infrastructure": "1.0.0.0",
		"Mvc3Futures":                "3.0.20105.0",
		"Ninject":                    "3.0.1.10",
		"Ninject.Web.Common":         "3.0.0.7",
		"WebActivator":               "1.5",
		"Microsoft.Net.Compilers":    "1.0.0",
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

	// Check development dependency scope
	if dep, ok := depMap["Microsoft.Net.Compilers"]; !ok {
		t.Error("expected Microsoft.Net.Compilers dependency")
	} else if dep.Scope != core.Development {
		t.Errorf("Microsoft.Net.Compilers scope = %v, want Development", dep.Scope)
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

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 5 packages with versions
	expected := map[string]string{
		"Argu":           "2.1",
		"Mono.Cecil":     "0.9.6.1",
		"Chessie":        "0.5.1",
		"FSharp.Core":    "4.0.0.1",
		"Newtonsoft.Json": "9.0.1",
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

func TestExampleNoVersionCsproj(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example-no-version.csproj")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &csprojParser{}
	deps, err := parser.Parse("example-no-version.csproj", content)
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

	// All 2 packages
	expected := map[string]string{
		"Microsoft.AspNetCore.App":          "",
		"Microsoft.AspNetCore.Razor.Design": "2.2.0",
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

func TestExampleUpdateCsproj(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example-update.csproj")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &csprojParser{}
	deps, err := parser.Parse("example-update.csproj", content)
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

	// All 2 packages
	expected := map[string]string{
		"Microsoft.AspNetCore":             "1.1.1",
		"Microsoft.AspNetCore.StaticFiles": "2.2.0",
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

func TestPackagesLockJson(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/packages.lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &packagesLockParser{}
	deps, err := parser.Parse("packages.lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 284 {
		t.Fatalf("expected 284 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"System.IO.Pipelines":                        "4.5.2",
		"System.Reflection.Metadata":                 "1.6.0",
		"Microsoft.AspNetCore.Http.Abstractions":    "2.2.0",
		"Microsoft.AspNetCore.Identity.UI":          "2.2.0",
		"Microsoft.EntityFrameworkCore.Design":       "2.2.0",
		"Microsoft.NETCore.Platforms":                "2.2.0",
		"System.IdentityModel.Tokens.Jwt":            "5.3.0",
		"Microsoft.NETCore.App":                      "2.2.0",
	}

	for name, wantVer := range samples {
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

func TestProjectLockJson(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/Project.lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &projectAssetsParser{}
	deps, err := parser.Parse("Project.lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 162 {
		t.Fatalf("expected 162 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"EntityFramework.InMemory":               "7.0.0-beta7",
		"System.ComponentModel.Annotations":     "4.0.11-beta-23225",
		"Microsoft.AspNet.Mvc.Cors":              "6.0.0-beta7",
		"Newtonsoft.Json":                        "6.0.6",
		"System.Diagnostics.Process":             "4.0.0-beta-23225",
		"Microsoft.AspNet.Hosting.Abstractions":  "1.0.0-beta7",
		"Microsoft.AspNet.Routing":               "1.0.0-beta7",
		"Microsoft.AspNet.StaticFiles":           "1.0.0-beta7",
	}

	for name, wantVer := range samples {
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

func TestProjectJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/Project.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &projectJSONParser{}
	deps, err := parser.Parse("Project.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 13 {
		t.Fatalf("expected 13 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All packages with expected versions
	expected := map[string]string{
		"Microsoft.AspNet.Server.Kestrel":                   "1.0.0-beta7",
		"Microsoft.AspNet.Server.IIS":                       "1.0.0-beta7",
		"Microsoft.AspNet.Mvc":                              "6.0.0-beta7",
		"Microsoft.AspNet.Server.WebListener":               "1.0.0-beta7",
		"Microsoft.AspNet.StaticFiles":                      "1.0.0-beta7",
		"EntityFramework.InMemory":                          "7.0.0-beta7",
		"EntityFramework.SqlServer":                         "7.0.0-beta7",
		"Microsoft.AspNet.Authentication.Cookies":           "1.0.0-beta7",
		"Microsoft.AspNet.Identity.EntityFramework":         "3.0.0-beta7",
		"Microsoft.Framework.Configuration":                 "1.0.0-beta7",
		"Microsoft.Framework.Configuration.EnvironmentVariables": "1.0.0-beta7",
		"Microsoft.Framework.Configuration.Json":            "1.0.0-beta7",
		"AutoMapper":                                        "4.0.0-alpha1",
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
		if !dep.Direct {
			t.Errorf("%s should be direct dependency", name)
		}
	}
}

func TestDepsJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example.deps.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &depsJSONParser{}
	deps, err := parser.Parse("example.deps.json", content)
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

	// All packages with expected versions
	expected := map[string]string{
		"Newtonsoft.Json":                     "13.0.1",
		"Microsoft.Extensions.DependencyInjection": "6.0.0",
		"Serilog":                             "2.10.0",
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

func TestCsprojReferences(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example-references-tag.csproj")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &csprojParser{}
	deps, err := parser.Parse("example-references-tag.csproj", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 3 non-system dependencies (System.Runtime.* is filtered as system)
	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All packages with expected versions
	expected := map[string]string{
		"FluentCommandLineParser":       "1.0.25.0",
		"log4net":                        "2.0.15.0",
		"Sequel.Core.Cryptography":       "1.0.0.0",
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

	// Verify system assemblies are excluded
	systemAssemblies := []string{"System", "System.Core", "System.Web", "Microsoft.CSharp", "WindowsBase", "PresentationCore", "System.Runtime.SomethingInternal"}
	for _, name := range systemAssemblies {
		if _, ok := depMap[name]; ok {
			t.Errorf("system assembly %s should be excluded", name)
		}
	}
}
