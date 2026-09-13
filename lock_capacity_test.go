package manifests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestLockDependencyCapacity(t *testing.T) {
	tests := []struct {
		name, content string
		want          []Dependency
	}{
		{"composer.lock", `{"packages":[{"name":"vendor/runtime","version":"1.2.3","dist":{"shasum":"abc","sha256":"def"}}],"packages-dev":[{"name":"vendor/dev","version":"2.0.0","dist":{"shasum":"abc"}}]}`, []Dependency{
			{Name: "vendor/runtime", Version: "1.2.3", Scope: Runtime, Integrity: "sha256-def", PURL: "pkg:composer/vendor/runtime@1.2.3"},
			{Name: "vendor/dev", Version: "2.0.0", Scope: Development, Integrity: "sha1-abc", PURL: "pkg:composer/vendor/dev@2.0.0"},
		}},
		{"packages.lock.json", `{"version":1,"dependencies":{"net8.0":{"Alpha":{"type":"Direct","resolved":"1.2.3","contentHash":"abc"}},"net9.0":{"Alpha":{"type":"Direct","resolved":"1.2.3","contentHash":"abc"},"Beta":{"type":"Transitive","resolved":"2.0.0"}},"net10.0":{"Gamma":{"type":"Transitive","resolved":"3.0.0"}}}}`, []Dependency{
			{Name: "Alpha", Version: "1.2.3", Scope: Runtime, Integrity: "sha512-abc", Direct: true, PURL: "pkg:nuget/Alpha@1.2.3"},
			{Name: "Beta", Version: "2.0.0", Scope: Runtime, PURL: "pkg:nuget/Beta@2.0.0"},
			{Name: "Gamma", Version: "3.0.0", Scope: Runtime, PURL: "pkg:nuget/Gamma@3.0.0"},
		}},
		{"project.assets.json", `{"targets":{"net8.0":{"Alpha/1.2.3":{"type":"package"},"App/1.0":{"type":"project"},"invalid":{"type":"package"}},"net9.0":{"Alpha/1.2.3":{"type":"package"},"Beta/2.0.0":{"type":"package"}},"net10.0":{"Gamma/3.0.0":{"type":"package"}}},"libraries":{"Alpha/1.2.3":{"sha512":"abc"}}}`, []Dependency{
			{Name: "Alpha", Version: "1.2.3", Scope: Runtime, Integrity: "sha512-abc", PURL: "pkg:nuget/Alpha@1.2.3"},
			{Name: "Beta", Version: "2.0.0", Scope: Runtime, PURL: "pkg:nuget/Beta@2.0.0"},
			{Name: "Gamma", Version: "3.0.0", Scope: Runtime, PURL: "pkg:nuget/Gamma@3.0.0"},
		}},
	}
	for _, name := range []string{"example.deps.json", "Project.lock.json"} {
		tests = append(tests, struct {
			name, content string
			want          []Dependency
		}{name, `{"libraries":{"Alpha/1.2.3":{"type":"package","sha512":"abc"},"Beta/2.0.0":{"type":"package"},"App/1.0":{"type":"project"},"invalid":{"type":"package"}}}`, []Dependency{
			{Name: "Alpha", Version: "1.2.3", Scope: Runtime, Integrity: "sha512-abc", PURL: "pkg:nuget/Alpha@1.2.3"},
			{Name: "Beta", Version: "2.0.0", Scope: Runtime, PURL: "pkg:nuget/Beta@2.0.0"},
		}})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Parse(tt.name, []byte(tt.content))
			if err != nil {
				t.Fatal(err)
			}
			sortDeps := func(deps []Dependency) {
				slices.SortFunc(deps, func(a, b Dependency) int {
					if a.Name < b.Name {
						return -1
					}
					if a.Name > b.Name {
						return 1
					}
					return 0
				})
			}
			sortDeps(result.Dependencies)
			sortDeps(tt.want)
			if !reflect.DeepEqual(result.Dependencies, tt.want) {
				t.Fatalf("got %#v, want %#v", result.Dependencies, tt.want)
			}
			for _, empty := range []string{`{}`, `{"packages":[],"packages-dev":[],"dependencies":{"net8.0":{}},"targets":{"net8.0":{"App/1.0":{"type":"project"},"invalid":{"type":"package"}}},"libraries":{"App/1.0":{"type":"project"},"invalid":{"type":"package"}}}`} {
				result, err = Parse(tt.name, []byte(empty))
				if err != nil {
					t.Fatal(err)
				}
				if result.Dependencies != nil {
					t.Fatalf("empty dependencies = %#v", result.Dependencies)
				}
			}
		})
	}
}

func lockCapacityInput(name string, count, frameworks int) []byte {
	packages := make([]map[string]any, 0, count)
	libraries := make(map[string]any, count)
	dependencies := make(map[string]any, count)
	targets := make(map[string]any, count)
	for i := range count {
		pkg := fmt.Sprintf("Package%d", i)
		packages = append(packages, map[string]any{"name": "vendor/" + pkg, "version": "1.2.3"})
		libraries[pkg+"/1.2.3"] = map[string]any{"type": "package", "sha512": "abc"}
		targets[pkg+"/1.2.3"] = map[string]any{"type": "package"}
		dependencies[pkg] = map[string]any{"type": "Direct", "resolved": "1.2.3", "contentHash": "abc"}
	}
	var input any
	switch name {
	case "composer.lock":
		input = map[string]any{"packages": packages[:count/2], "packages-dev": packages[count/2:]}
	case "packages.lock.json", "project.assets.json":
		groups := make(map[string]any, frameworks)
		for i := range frameworks {
			if name == "packages.lock.json" {
				groups[fmt.Sprintf("net%d.0", 8+i)] = dependencies
			} else {
				groups[fmt.Sprintf("net%d.0", 8+i)] = targets
			}
		}
		if name == "packages.lock.json" {
			input = map[string]any{"version": 1, "dependencies": groups}
		} else {
			input = map[string]any{"targets": groups, "libraries": libraries}
		}
	default:
		input = map[string]any{"libraries": libraries}
	}
	content, err := json.Marshal(input)
	if err != nil {
		panic(err)
	}
	return content
}

func BenchmarkLockCapacity(b *testing.B) {
	for _, name := range []string{"composer.lock", "packages.lock.json", "project.assets.json", "example.deps.json", "Project.lock.json"} {
		for _, count := range []int{2, 100, 10000} {
			groups := []int{1}
			if name == "packages.lock.json" || name == "project.assets.json" {
				groups = append(groups, 3)
			}
			for _, frameworks := range groups {
				b.Run(fmt.Sprintf("%s/%d/%d", name, count, frameworks), func(b *testing.B) {
					content := lockCapacityInput(name, count, frameworks)
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						result, err := Parse(name, content)
						if err != nil {
							b.Fatal(err)
						}
						if len(result.Dependencies) != count {
							b.Fatalf("got %d dependencies, want %d", len(result.Dependencies), count)
						}
					}
				})
			}
		}
	}
}

func BenchmarkLockCapacityFixtures(b *testing.B) {
	for _, path := range []string{"composer/composer.lock", "nuget/packages.lock.json", "nuget/nuget_project.assets.json", "nuget/example.deps.json", "nuget/Project.lock.json"} {
		b.Run(path, func(b *testing.B) {
			content, err := os.ReadFile("testdata/" + path)
			if err != nil {
				b.Fatal(err)
			}
			name := filepath.Base(path)
			if name == "nuget_project.assets.json" {
				name = "project.assets.json"
			}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				result, err := Parse(name, content)
				if err != nil {
					b.Fatal(err)
				}
				if len(result.Dependencies) == 0 {
					b.Fatal("no dependencies")
				}
			}
		})
	}
}
