package manifests

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestIdentify(t *testing.T) {
	tests := []struct {
		filename  string
		ecosystem string
		kind      Kind
		ok        bool
	}{
		{"package.json", "npm", Manifest, true},
		{"package-lock.json", "npm", Lockfile, true},
		{"npm-shrinkwrap.json", "npm", Lockfile, true},
		{"unknown.txt", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			eco, kind, ok := Identify(tt.filename)
			if ok != tt.ok {
				t.Errorf("ok = %v, want %v", ok, tt.ok)
			}
			if eco != tt.ecosystem {
				t.Errorf("ecosystem = %q, want %q", eco, tt.ecosystem)
			}
			if kind != tt.kind {
				t.Errorf("kind = %q, want %q", kind, tt.kind)
			}
		})
	}
}

func TestPURLGeneration(t *testing.T) {
	content, err := os.ReadFile("testdata/npm/package.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	result, err := Parse("package.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	deps := make(map[string]Dependency)
	for _, d := range result.Dependencies {
		deps[d.Name] = d
	}

	// Check PURL generation (no version for manifests)
	if babel, ok := deps["babel"]; ok {
		if babel.PURL != "pkg:npm/babel" {
			t.Errorf("babel PURL = %q, want %q", babel.PURL, "pkg:npm/babel")
		}
	}

	// Check PURL for scoped package (@ is URL-encoded in PURL)
	if actual, ok := deps["@some-scope/actual-package"]; ok {
		if actual.PURL != "pkg:npm/%40some-scope/actual-package" {
			t.Errorf("alias PURL = %q, want %q", actual.PURL, "pkg:npm/%40some-scope/actual-package")
		}
	}

	// Check lockfile PURL includes version
	lockContent, err := os.ReadFile("testdata/npm/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	lockResult, err := Parse("package-lock.json", lockContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for _, d := range lockResult.Dependencies {
		if d.Name == "express" {
			if d.PURL != "pkg:npm/express@4.15.3" {
				t.Errorf("express PURL = %q, want %q", d.PURL, "pkg:npm/express@4.15.3")
			}
			break
		}
	}
}

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

func TestNPMV2LockfileCRLF(t *testing.T) {
	content, err := os.ReadFile("testdata/npm/npm-lockfile-version-2/package-lock.json")
	if err != nil {
		t.Fatal(err)
	}
	lf := bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
	crlf := bytes.ReplaceAll(lf, []byte("\n"), []byte("\r\n"))
	for _, filename := range []string{"package-lock.json", "npm-shrinkwrap.json"} {
		t.Run(filename, func(t *testing.T) {
			want, err := Parse(filename, lf)
			if err != nil {
				t.Fatal(err)
			}
			if len(want.Dependencies) != 3 {
				t.Fatalf("LF fixture has %d dependencies, want 3", len(want.Dependencies))
			}
			got, err := Parse(filename, crlf)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("CRLF result differs from LF: got %d dependencies, want %d", len(got.Dependencies), len(want.Dependencies))
			}
		})
	}
}
