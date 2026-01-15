package alpine

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestAPKBUILD(t *testing.T) {
	content, err := os.ReadFile("../../testdata/alpine/APKBUILD")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &apkbuildParser{}
	deps, err := parser.Parse("APKBUILD", content)
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

	// All 13 packages with versions and scopes
	// Note: python3 appears twice (build and test), map keeps last
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"ca-certificates-bundle", "", core.Runtime},
		{"brotli-dev", "", core.Development},
		{"c-ares-dev", "", core.Development},
		{"libidn2-dev", "", core.Development},
		{"libpsl-dev", "", core.Development},
		{"nghttp2-dev", "", core.Development},
		{"openssl-dev", ">3", core.Development},
		{"zlib-dev", "", core.Development},
		{"zstd-dev", "", core.Development},
		{"perl", "", core.Build},
		{"nghttp2", "", core.Test},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}

	// python3 appears twice (build and test), verify it exists
	if _, ok := depMap["python3"]; !ok {
		t.Error("expected python3 dependency")
	}
}

func TestAPKBUILDWithVersions(t *testing.T) {
	content, err := os.ReadFile("../../testdata/alpine/APKBUILD-with-versions")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &apkbuildParser{}
	deps, err := parser.Parse("APKBUILD", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 9 {
		t.Fatalf("expected 9 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 9 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"libfoo", ">=1.0", core.Runtime},
		{"libbar", "<2.0", core.Runtime},
		{"libbaz", "", core.Runtime},
		{"gcc", "", core.Build},
		{"make", "", core.Build},
		{"openssl-dev", ">3", core.Build},
		{"zlib-dev", ">=1.2.3", core.Build},
		{"pytest", "", core.Test},
		{"python3", ">=3.9", core.Test},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}
