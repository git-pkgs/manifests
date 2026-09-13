package manifests

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestNPMLockfileNestedDependencies(t *testing.T) {
	const content = `{"lockfileVersion":1,"dependencies":{
		"parent":{"version":"1.0.0","dev":true,"dependencies":{
			"shared":{"version":"2.0.0","optional":true,"dependencies":{
				"@scope/leaf":{"version":"3.0.0","dev":true,"optional":true,"integrity":"sha512-leaf"}
			}}
		}},
		"shared":{"version":"4.0.0"}
	}}`
	result, err := Parse("package-lock.json", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	want := []Dependency{
		{Name: "@scope/leaf", Version: "3.0.0", Scope: Development, Integrity: "sha512-leaf", PURL: "pkg:npm/%40scope/leaf@3.0.0"},
		{Name: "parent", Version: "1.0.0", Scope: Development, PURL: "pkg:npm/parent@1.0.0"},
		{Name: "shared", Version: "2.0.0", Scope: Optional, PURL: "pkg:npm/shared@2.0.0"},
		{Name: "shared", Version: "4.0.0", Scope: Runtime, PURL: "pkg:npm/shared@4.0.0"},
	}
	sort.Slice(result.Dependencies, func(i, j int) bool {
		a, b := result.Dependencies[i], result.Dependencies[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		return a.Version < b.Version
	})
	if !reflect.DeepEqual(result.Dependencies, want) {
		t.Fatalf("got %+v, want %+v", result.Dependencies, want)
	}
}

func TestNPMLockfileLineIteration(t *testing.T) {
	const content = `{
  "lockfileVersion": 3,
  "packages": {
    "": {
      "dependencies": {"direct": "^1.0.0"}
    },
    "node_modules/direct": {
      "version": "1.0.0",
      "integrity": "LONG_INTEGRITY"
    },
    "node_modules/direct/node_modules/@scope/nested": {
      "version": "2.0.0",
      "devOptional": true
    },
    "node_modules/last": {
      "version": "3.0.0",
      "optional": true
    }
  }
}`
	integrity := strings.Repeat("x", 128<<10)
	for _, version := range []int{2, 3} {
		for _, newline := range []string{"\n", "\r\n"} {
			for _, trailing := range []bool{false, true} {
				t.Run(fmt.Sprintf("v%d/newline=%q/trailing=%t", version, newline, trailing), func(t *testing.T) {
					input := strings.Replace(content, `"lockfileVersion": 3`, fmt.Sprintf(`"lockfileVersion": %d`, version), 1)
					input = strings.Replace(input, "LONG_INTEGRITY", integrity, 1)
					input = strings.ReplaceAll(input, "\n", newline)
					if trailing {
						input += newline
					}
					result, err := Parse("npm-shrinkwrap.json", []byte(input))
					if err != nil {
						t.Fatal(err)
					}
					want := []Dependency{
						{Name: "direct", Version: "1.0.0", Scope: Runtime, Direct: true, Integrity: integrity, PURL: "pkg:npm/direct@1.0.0"},
						{Name: "@scope/nested", Version: "2.0.0", Scope: Development, PURL: "pkg:npm/%40scope/nested@2.0.0"},
						{Name: "last", Version: "3.0.0", Scope: Optional, PURL: "pkg:npm/last@3.0.0"},
					}
					if !reflect.DeepEqual(result.Dependencies, want) {
						t.Fatal("dependency fields or order differ")
					}
				})
			}
		}
	}
}

func TestNPMLockfileEmptyDependencies(t *testing.T) {
	for _, content := range []string{
		`{"lockfileVersion":1,"dependencies":{}}`,
		"{\n  \"lockfileVersion\": 3,\n  \"packages\": {\n  }\n}",
		"{\n  \"lockfileVersion\": 3,\n  \"packages\": {\n    \"node_modules/\": {\n      \"version\": \"1.0.0\"\n    }\n  }\n}",
	} {
		result, err := Parse("package-lock.json", []byte(content))
		if err != nil {
			t.Fatal(err)
		}
		if result.Dependencies != nil {
			t.Fatalf("expected nil dependencies, got %+v", result.Dependencies)
		}
	}
}
