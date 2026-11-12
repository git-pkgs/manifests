package manifests

import (
	"os"
	"testing"
)

func TestSwiftTargetScopes(t *testing.T) {
	result := parseSwiftFixture(t, "testdata/swift/usage/Package.swift")
	want := map[string]Scope{
		"github.com/Example/Runtime.Kit": Runtime,
		"example.com/acme/Test.Kit":      Test,
		"example.com/acme/Generator":     Build,
		"example.com/acme/Plugin.Kit":    Build,
		"vendor.logging":                 Test,
		"example.com/acme/Shared.Kit":    Runtime,
		"example.com/acme/Unused.Kit":    Runtime,
	}
	if result.Name != "ExampleApp" || len(result.Dependencies) != len(want) {
		t.Fatalf("Parse = %+v", result)
	}
	for _, dep := range result.Dependencies {
		scope, ok := want[dep.Name]
		if !ok || dep.Scope != scope || !dep.Direct {
			t.Errorf("dependency = %+v, want direct dependency with scope %q", dep, scope)
		}
		if dep.Name == "vendor.logging" && (dep.PURL != "pkg:swift/vendor/logging" || dep.Source.Kind != SourceRegistry || dep.Source.Value != "vendor.Logging") {
			t.Errorf("registry identity was changed or mapped to a repository: %+v", dep)
		}
		if dep.Name == "github.com/Example/Runtime.Kit" && dep.Source.Value != "https://GitHub.com/Example/Runtime.Kit.git" {
			t.Errorf("source spelling changed: %+v", dep)
		}
	}
}

func TestSwiftLockfileDoesNotDeclareTransitivePackages(t *testing.T) {
	manifest := parseSwiftFixture(t, "testdata/swift/transitive/Package.swift")
	upstream := parseSwiftFixture(t, "testdata/swift/transitive/upstream/Package.swift")
	lockfile := parseSwiftFixture(t, "testdata/swift/transitive/Package.resolved")
	if len(manifest.Dependencies) != 1 || manifest.Dependencies[0].Name != "example.com/acme/Runtime.Kit" || !manifest.Dependencies[0].Direct {
		t.Fatalf("manifest dependencies = %+v", manifest.Dependencies)
	}
	if len(upstream.Dependencies) != 1 || upstream.Dependencies[0].Name != "example.com/acme/Support.Kit" {
		t.Fatalf("upstream dependencies = %+v", upstream.Dependencies)
	}
	if lockfile.Kind != Lockfile || len(lockfile.Dependencies) != 2 {
		t.Fatalf("lockfile = %+v", lockfile)
	}
	foundTransitive := false
	for _, dep := range lockfile.Dependencies {
		if dep.Direct {
			t.Errorf("lockfile pin became a direct dependency: %+v", dep)
		}
		if dep.Name == "example.com/acme/Support.Kit" {
			foundTransitive = true
		}
	}
	if !foundTransitive {
		t.Fatal("lockfile does not contain the transitive Support.Kit dependency")
	}
}

func TestSwiftScopeFallbacks(t *testing.T) {
	tests := []struct {
		name    string
		targets string
		want    Scope
	}{
		{"mixed build and test", `.plugin(name: "Build", capability: .buildTool(), dependencies: [.product(name: "Tool", package: "tools")]),
			.testTarget(name: "Tests", dependencies: [.product(name: "Tool", package: "tools")])`, Build},
		{"macro", `.macro(name: "Macros", dependencies: [.product(name: "Tool", package: "tools")])`, Build},
		{"test plugin usage", `.testTarget(name: "Tests", plugins: [.plugin(name: "Tool", package: "tools")])`, Build},
		{"unqualified product", `.target(name: "App", dependencies: [.product(name: "MaybeTools")]),
			.testTarget(name: "Tests", dependencies: [.product(name: "Tool", package: "tools")])`, Runtime},
		{"unknown product string", `.target(name: "App", dependencies: ["MaybeTools"]),
			.testTarget(name: "Tests", dependencies: [.product(name: "Tool", package: "tools")])`, Runtime},
		{"computed dependencies", `.target(name: "App", dependencies: runtimeDependencies),
			.testTarget(name: "Tests", dependencies: [.product(name: "Tool", package: "tools")])`, Runtime},
		{"ordinary helper", `.target(name: "TestHelper", dependencies: [.product(name: "Tool", package: "tools")]),
			.testTarget(name: "Tests", dependencies: ["TestHelper"])`, Runtime},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `let package = Package(name: "App",
				dependencies: [.package(url: "https://example.com/acme/Tools.git", from: "1.0.0")],
				targets: [` + tt.targets + `])`
			result, err := Parse("Package.swift", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Dependencies) != 1 || result.Dependencies[0].Scope != tt.want {
				t.Fatalf("Dependencies = %+v, want scope %q", result.Dependencies, tt.want)
			}
		})
	}
}

func TestSwiftSourceAndRegistryIdentitiesStaySeparate(t *testing.T) {
	tests := []struct {
		filename string
		content  string
		wantPURL string
	}{
		{"Package.swift", `let package = Package(name: "App", dependencies: [
			.package(url: "https://GitHub.com/Example/Logging.Kit.git", exact: "1.0.0"),
			.package(id: "Example.Logging-Kit", exact: "1.0.0")
		])`, "pkg:swift/example/logging-kit"},
		{"Package.resolved", `{"version":2,"pins":[
			{"identity":"logging.kit","kind":"remoteSourceControl","location":"https://GitHub.com/Example/Logging.Kit.git","state":{"version":"1.0.0"}},
			{"identity":"Example.Logging-Kit","kind":"registry","location":"Example.Logging-Kit","state":{"version":"1.0.0"}}
		]}`, "pkg:swift/example/logging-kit@1.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result, err := Parse(tt.filename, []byte(tt.content))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Dependencies) != 2 {
				t.Fatalf("Dependencies = %+v", result.Dependencies)
			}
			source, registry := result.Dependencies[0], result.Dependencies[1]
			if source.Name != "github.com/Example/Logging.Kit" || source.Source.Kind != SourceGit || source.Source.Value != "https://GitHub.com/Example/Logging.Kit.git" {
				t.Errorf("source identity = %+v", source)
			}
			if registry.Name != "example.logging-kit" || registry.Source.Kind != SourceRegistry || registry.Source.Value != "Example.Logging-Kit" {
				t.Errorf("registry identity = %+v", registry)
			}
			if registry.PURL != tt.wantPURL || registry.RegistryURL != "" {
				t.Errorf("registry dependency = %+v, want PURL %q and no registry URL", registry, tt.wantPURL)
			}
		})
	}
}

func parseSwiftFixture(t *testing.T, filename string) *ParseResult {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Parse(filename, content)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
