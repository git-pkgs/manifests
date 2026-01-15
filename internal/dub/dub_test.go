package dub

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestDubJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/dub/dub.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &dubJSONParser{}
	deps, err := parser.Parse("dub.json", content)
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

	// Check normal dependency with string version
	if dep, ok := depMap["vibe-d"]; !ok {
		t.Error("expected vibe-d dependency")
	} else {
		if dep.Version != "~>0.7.22" {
			t.Errorf("expected vibe-d version ~>0.7.22, got %s", dep.Version)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("expected vibe-d scope Runtime, got %v", dep.Scope)
		}
	}

	// Check optional dependency with object
	if dep, ok := depMap["libdparse"]; !ok {
		t.Error("expected libdparse dependency")
	} else {
		if dep.Version != "~>0.2.0" {
			t.Errorf("expected libdparse version ~>0.2.0, got %s", dep.Version)
		}
		if dep.Scope != core.Optional {
			t.Errorf("expected libdparse scope Optional, got %v", dep.Scope)
		}
	}
}

func TestDubSDL(t *testing.T) {
	content, err := os.ReadFile("../../testdata/dub/dub.sdl")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &dubSDLParser{}
	deps, err := parser.Parse("dub.sdl", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.Name != "vibe-d" {
		t.Errorf("expected vibe-d, got %s", dep.Name)
	}
	if dep.Version != "~>0.7.23" {
		t.Errorf("expected version ~>0.7.23, got %s", dep.Version)
	}
}
