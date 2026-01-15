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

	// Check runtime dependency
	if dep, ok := depMap["analyzer"]; !ok {
		t.Error("expected analyzer dependency")
	} else {
		if dep.Version != ">=0.22.0 <0.25.0" {
			t.Errorf("expected analyzer version >=0.22.0 <0.25.0, got %s", dep.Version)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("expected analyzer scope Runtime, got %v", dep.Scope)
		}
	}

	// Check dev dependency
	if dep, ok := depMap["benchmark_harness"]; !ok {
		t.Error("expected benchmark_harness dependency")
	} else {
		if dep.Scope != core.Development {
			t.Errorf("expected benchmark_harness scope Development, got %v", dep.Scope)
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

	// Check analyzer
	if dep, ok := depMap["analyzer"]; !ok {
		t.Error("expected analyzer dependency")
	} else if dep.Version != "0.24.6" {
		t.Errorf("expected analyzer version 0.24.6, got %s", dep.Version)
	}

	// Check args
	if dep, ok := depMap["args"]; !ok {
		t.Error("expected args dependency")
	} else if dep.Version != "0.12.2+6" {
		t.Errorf("expected args version 0.12.2+6, got %s", dep.Version)
	}
}
