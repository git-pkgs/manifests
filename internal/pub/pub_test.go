package pub

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPubspecYAML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pub/pubspec.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pubspecYAMLParser{}
	deps, err := parser.Parse("pubspec.yaml", content)
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

	// All 4 packages with versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"analyzer":          {">=0.22.0 <0.25.0", core.Runtime},
		"args":              {">=0.12.0 <0.13.0", core.Runtime},
		"benchmark_harness": {">=1.0.0 <2.0.0", core.Development},
		"guinness":          {">=0.1.9 <0.2.0", core.Development},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
		}
	}
}

func TestPubspecLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pub/pubspec.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pubspecLockParser{}
	deps, err := parser.Parse("pubspec.lock", content)
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

	// All 4 packages with versions
	expected := map[string]string{
		"analyzer": "0.24.6",
		"args":     "0.12.2+6",
		"barback":  "0.15.2+7",
		"which":    "0.1.3",
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
