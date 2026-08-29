package nuget

import (
	"net/url"
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

const testVersion100 = "1.0.0"

// assertParseDeps reads a fixture, parses it, checks the dependency count,
// and verifies each expected name/version pair is present.
func assertParseDeps(t *testing.T, fixturePath string, parser core.Parser, parseFilename string, wantCount int, expected map[string]string) map[string]core.Dependency {
	t.Helper()

	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	res, err := parser.Parse(parseFilename, content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != wantCount {
		t.Fatalf("expected %d dependencies, got %d", wantCount, len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
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

	return depMap
}

func assertDependencyIntegrity(t *testing.T, deps map[string]core.Dependency, name, want string) {
	t.Helper()

	dep, ok := deps[name]
	if !ok {
		t.Fatalf("expected %s dependency", name)
	}
	if dep.Integrity != want {
		t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, want)
	}
}

func TestCsproj(t *testing.T) {
	assertParseDeps(t, "../../testdata/nuget/example.csproj", &csprojParser{}, "example.csproj", 8, map[string]string{
		"Microsoft.AspNetCore":                     "1.1.1",
		"Microsoft.AspNetCore.Mvc":                 "1.1.2",
		"Microsoft.AspNetCore.StaticFiles":         "1.1.1",
		"Microsoft.Extensions.Logging.Debug":       "1.1.1",
		"Microsoft.Extensions.DependencyInjection": "1.1.1",
		"Microsoft.VisualStudio.Web.BrowserLink":   "1.1.0",
		"System.Resources.Extensions":              "4.7.0",
		"Contoso.Utility.UsefulStuff":              "3.6.0",
	})
}

func TestCsprojDeclarations(t *testing.T) {
	content := []byte(`<Project>
  <ItemGroup>
    <PackageReference Include="Example" Version="1.0.0" />
    <PackageReference Include="example"><VersionOverride>2.0.0</VersionOverride></PackageReference>
    <PackageReference Update="Central" VersionOverride="3.0.0" />
  </ItemGroup>
  <ItemGroup Condition="'$(TargetFramework)' == 'net8.0'">
    <PackageReference Include="Conditional" Version="4.0.0" Condition="'$(Configuration)' == 'Debug'" />
  </ItemGroup>
</Project>`)

	result, err := (&csprojParser{}).Parse("example.csproj", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	condition := url.PathEscape("'$(TargetFramework)' == 'net8.0'")
	itemCondition := url.PathEscape("'$(Configuration)' == 'Debug'")
	want := map[string]string{
		"package-references/example":   "1.0.0",
		"package-references/example/2": "2.0.0",
		"package-references/central":   "3.0.0",
		"package-references/" + condition + "/" + itemCondition + "/conditional": "4.0.0",
	}
	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		if version, ok := want[declaration.Location]; !ok || declaration.Version != version || !declaration.Direct {
			t.Errorf("unexpected declaration: %+v", declaration)
		}
	}
	if len(result.Dependencies) != 3 {
		t.Fatalf("Dependencies has %d entries, want 3: %+v", len(result.Dependencies), result.Dependencies)
	}
	dependencyVersions := make(map[string]string)
	for _, dependency := range result.Dependencies {
		dependencyVersions[dependency.Name] = dependency.Version
	}
	if dependencyVersions["Example"] != "1.0.0" ||
		dependencyVersions["example"] != "" ||
		dependencyVersions["Conditional"] != "4.0.0" {
		t.Errorf("Dependencies changed existing PackageReference versions: %+v", result.Dependencies)
	}
}

func TestCentralPackagesDeclarations(t *testing.T) {
	content := []byte(`<Project>
  <ItemGroup>
    <PackageVersion Include="Newtonsoft.Json" Version="13.0.3" />
    <PackageVersion Update="Serilog"><Version>4.0.0</Version></PackageVersion>
    <PackageVersion Include="Conditional" Version="5.0.0" Condition="'$(TargetFramework)' == 'net8.0'" />
  </ItemGroup>
</Project>`)

	result, err := (&centralPackagesParser{}).Parse("Directory.Packages.props", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := map[string]string{
		"package-versions/newtonsoft.json": "13.0.3",
		"package-versions/serilog":         "4.0.0",
		"package-versions/" + url.PathEscape("'$(TargetFramework)' == 'net8.0'") + "/conditional": "5.0.0",
	}
	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		if version, ok := want[declaration.Location]; !ok || declaration.Version != version || !declaration.Direct {
			t.Errorf("unexpected declaration: %+v", declaration)
		}
	}
	if len(result.Dependencies) != 0 {
		t.Fatalf("PackageVersion catalog entries should not widen Dependencies: %+v", result.Dependencies)
	}
}

func TestCentralPackagesGlobalReferences(t *testing.T) {
	content := []byte(`<Project>
  <ItemGroup Condition="'$(Configuration)' == 'Debug'">
    <GlobalPackageReference Include="Nerdbank.GitVersioning" Version="3.5.119" />
  </ItemGroup>
</Project>`)

	result, err := (&centralPackagesParser{}).Parse("Directory.Packages.props", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(result.Declarations) != 1 {
		t.Fatalf("Declarations has %d entries, want 1: %+v", len(result.Declarations), result.Declarations)
	}
	condition := url.PathEscape("'$(Configuration)' == 'Debug'")
	declaration := result.Declarations[0]
	if declaration.Location != "global-package-references/"+condition+"/nerdbank.gitversioning" ||
		declaration.Name != "Nerdbank.GitVersioning" || declaration.Version != "3.5.119" ||
		declaration.Scope != core.Development || !declaration.Direct {
		t.Errorf("unexpected declaration: %+v", declaration)
	}
	if len(result.Dependencies) != 1 {
		t.Fatalf("Dependencies has %d entries, want 1: %+v", len(result.Dependencies), result.Dependencies)
	}
	dependency := result.Dependencies[0]
	if dependency.Name != "Nerdbank.GitVersioning" || dependency.Version != "3.5.119" ||
		dependency.Scope != core.Development || !dependency.Direct {
		t.Errorf("unexpected dependency: %+v", dependency)
	}
}

func TestNuspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example.nuspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &nuspecParser{}
	res, err := parser.Parse("example.nuspec", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
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

func TestNuspecDeclarationsPreserveGroups(t *testing.T) {
	content := []byte(`<package><metadata><dependencies>
  <dependency id="Example" version="[1.0.0]" />
  <group targetFramework="net8.0"><dependency id="example" version="[2.0.0]" /></group>
  <group><dependency id="Any" version="[3.0.0]" /></group>
</dependencies></metadata></package>`)

	result, err := (&nuspecParser{}).Parse("example.nuspec", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := map[string]string{
		"dependencies/example":             "[1.0.0]",
		"dependency-groups/net8.0/example": "[2.0.0]",
		"dependency-groups/any":            "[3.0.0]",
	}
	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		if version, ok := want[declaration.Location]; !ok || declaration.Version != version {
			t.Errorf("unexpected declaration: %+v", declaration)
		}
	}
	if len(result.Dependencies) != 3 {
		t.Fatalf("Dependencies has %d entries, want 3: %+v", len(result.Dependencies), result.Dependencies)
	}
}

func TestPackagesConfig(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/packages.config")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &packagesConfigParser{}
	res, err := parser.Parse("packages.config", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(res.Dependencies))
	}
	if len(res.Declarations) != 7 {
		t.Fatalf("expected 7 declarations, got %d: %+v", len(res.Declarations), res.Declarations)
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All 7 packages with exact versions
	expected := map[string]string{
		"AutoMapper":                   "2.1.267",
		"Microsoft.Web.Infrastructure": "1.0.0.0",
		"Mvc3Futures":                  "3.0.20105.0",
		"Ninject":                      "3.0.1.10",
		"Ninject.Web.Common":           "3.0.0.7",
		"WebActivator":                 "1.5",
		"Microsoft.Net.Compilers":      testVersion100,
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
	res, err := parser.Parse("project.assets.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// Check package a
	if dep, ok := depMap["a"]; !ok {
		t.Error("expected a dependency")
	} else {
		if dep.Version != testVersion100 {
			t.Errorf("a version = %q, want %q", dep.Version, testVersion100)
		}
		wantIntegrity := "sha512-L3W3kgOOU5+2Tdtnzywcs4/a3XFbwcM7Ghvr2uWnhLUvBithluWlGI+0/lXFrDysXaRMLSRJdExSLuSJJQYuTg=="
		if dep.Integrity != wantIntegrity {
			t.Errorf("a integrity = %q, want %q", dep.Integrity, wantIntegrity)
		}
	}

	// Check package b
	if dep, ok := depMap["b"]; !ok {
		t.Error("expected b dependency")
	} else if dep.Version != testVersion100 {
		t.Errorf("b version = %q, want %q", dep.Version, testVersion100)
	}

	// Check package c from net2.2 framework
	if dep, ok := depMap["c"]; !ok {
		t.Error("expected c dependency")
	} else if dep.Version != testVersion100 {
		t.Errorf("c version = %q, want %q", dep.Version, testVersion100)
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
	res, err := parser.Parse("paket.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All 5 packages with versions
	expected := map[string]string{
		"Argu":            "2.1",
		"Mono.Cecil":      "0.9.6.1",
		"Chessie":         "0.5.1",
		"FSharp.Core":     "4.0.0.1",
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
	assertParseDeps(t, "../../testdata/nuget/example-no-version.csproj", &csprojParser{}, "example-no-version.csproj", 2, map[string]string{
		"Microsoft.AspNetCore.App":          "",
		"Microsoft.AspNetCore.Razor.Design": "2.2.0",
	})
}

func TestExampleUpdateCsproj(t *testing.T) {
	assertParseDeps(t, "../../testdata/nuget/example-update.csproj", &csprojParser{}, "example-update.csproj", 2, map[string]string{
		"Microsoft.AspNetCore":             "1.1.1",
		"Microsoft.AspNetCore.StaticFiles": "2.2.0",
	})
}

func TestPackagesLockJson(t *testing.T) {
	deps := assertParseDeps(t, "../../testdata/nuget/packages.lock.json", &packagesLockParser{}, "packages.lock.json", 284, map[string]string{
		"System.IO.Pipelines":                    "4.5.2",
		"System.Reflection.Metadata":             "1.6.0",
		"Microsoft.AspNetCore.Http.Abstractions": "2.2.0",
		"Microsoft.AspNetCore.Identity.UI":       "2.2.0",
		"Microsoft.EntityFrameworkCore.Design":   "2.2.0",
		"Microsoft.NETCore.Platforms":            "2.2.0",
		"System.IdentityModel.Tokens.Jwt":        "5.3.0",
		"Microsoft.NETCore.App":                  "2.2.0",
	})

	wantIntegrity := "sha512-L3W3kgOOU5+2Tdtnzywcs4/a3XFbwcM7Ghvr2uWnhLUvBithluWlGI+0/lXFrDysXaRMLSRJdExSLuSJJQYuTg=="
	assertDependencyIntegrity(t, deps, "Microsoft.AspNetCore.App", wantIntegrity)
}

func TestProjectLockJson(t *testing.T) {
	deps := assertParseDeps(t, "../../testdata/nuget/Project.lock.json", &projectLockJSONParser{}, "Project.lock.json", 162, map[string]string{
		"EntityFramework.InMemory":              "7.0.0-beta7",
		"System.ComponentModel.Annotations":     "4.0.11-beta-23225",
		"Microsoft.AspNet.Mvc.Cors":             "6.0.0-beta7",
		"Newtonsoft.Json":                       "6.0.6",
		"System.Diagnostics.Process":            "4.0.0-beta-23225",
		"Microsoft.AspNet.Hosting.Abstractions": "1.0.0-beta7",
		"Microsoft.AspNet.Routing":              "1.0.0-beta7",
		"Microsoft.AspNet.StaticFiles":          "1.0.0-beta7",
	})

	wantIntegrity := "sha512-rvlGuIXTu1pF9NfbCaK6ocDrP9iCRJ8UXfUs5IvU/vfjs/SobQEN+b3b/L7SpqLRL5/glsSSvPDX2wUOTNrOfA=="
	assertDependencyIntegrity(t, deps, "AutoMapper", wantIntegrity)
}

func TestProjectJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/Project.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &projectJSONParser{}
	res, err := parser.Parse("Project.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 13 {
		t.Fatalf("expected 13 dependencies, got %d", len(res.Dependencies))
	}
	if len(res.Declarations) != 13 {
		t.Fatalf("expected 13 declarations, got %d: %+v", len(res.Declarations), res.Declarations)
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All packages with expected versions
	expected := map[string]string{
		"Microsoft.AspNet.Server.Kestrel":                        "1.0.0-beta7",
		"Microsoft.AspNet.Server.IIS":                            "1.0.0-beta7",
		"Microsoft.AspNet.Mvc":                                   "6.0.0-beta7",
		"Microsoft.AspNet.Server.WebListener":                    "1.0.0-beta7",
		"Microsoft.AspNet.StaticFiles":                           "1.0.0-beta7",
		"EntityFramework.InMemory":                               "7.0.0-beta7",
		"EntityFramework.SqlServer":                              "7.0.0-beta7",
		"Microsoft.AspNet.Authentication.Cookies":                "1.0.0-beta7",
		"Microsoft.AspNet.Identity.EntityFramework":              "3.0.0-beta7",
		"Microsoft.Framework.Configuration":                      "1.0.0-beta7",
		"Microsoft.Framework.Configuration.EnvironmentVariables": "1.0.0-beta7",
		"Microsoft.Framework.Configuration.Json":                 "1.0.0-beta7",
		"AutoMapper":                                             "4.0.0-alpha1",
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
	res, err := parser.Parse("example.deps.json", content)
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

	// All packages with expected versions
	expected := map[string]string{
		"Newtonsoft.Json":                          "13.0.1",
		"Microsoft.Extensions.DependencyInjection": "6.0.0",
		"Serilog": "2.10.0",
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

	wantIntegrity := "sha512-ppPFpBcvxdsfUon7g7o+4/7SQXUz0MgZH74+YbS3cJVQ="
	assertDependencyIntegrity(t, depMap, "Newtonsoft.Json", wantIntegrity)
}

func TestCsprojReferences(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nuget/example-references-tag.csproj")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &csprojParser{}
	res, err := parser.Parse("example-references-tag.csproj", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 3 non-system dependencies (System.Runtime.* is filtered as system)
	if len(res.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All packages with expected versions
	expected := map[string]string{
		"FluentCommandLineParser":  "1.0.25.0",
		"log4net":                  "2.0.15.0",
		"Sequel.Core.Cryptography": "1.0.0.0",
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
