package manifests

import (
	"os"
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

	// Check PURL for scoped package
	if actual, ok := deps["@some-scope/actual-package"]; ok {
		if actual.PURL != "pkg:npm/some-scope/actual-package" {
			t.Errorf("alias PURL = %q, want %q", actual.PURL, "pkg:npm/some-scope/actual-package")
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
