package elm

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestElmPackageJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/elm/elm-package.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &elmPackageJSONParser{}
	deps, err := parser.Parse("elm-package.json", content)
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

	// Check elm-markdown with version range
	if dep, ok := depMap["evancz/elm-markdown"]; !ok {
		t.Error("expected evancz/elm-markdown dependency")
	} else {
		if dep.Version != "1.1.0 <= v < 2.0.0" {
			t.Errorf("expected version 1.1.0 <= v < 2.0.0, got %s", dep.Version)
		}
		if !dep.Direct {
			t.Error("expected direct dependency")
		}
	}

	// Check elm-lang/core
	if dep, ok := depMap["elm-lang/core"]; !ok {
		t.Error("expected elm-lang/core dependency")
	} else if dep.Version != "1.0.0 <= v < 2.0.0" {
		t.Errorf("expected version 1.0.0 <= v < 2.0.0, got %s", dep.Version)
	}
}

func TestElmLegacyJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/elm/elm_dependencies.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	// Legacy format is similar to elm-package.json
	parser := &elmPackageJSONParser{}
	deps, err := parser.Parse("elm-package.json", content)
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

	if dep, ok := depMap["johnpmayer/elm-webgl"]; !ok {
		t.Error("expected johnpmayer/elm-webgl dependency")
	} else if dep.Version != "0.1.1" {
		t.Errorf("expected version 0.1.1, got %s", dep.Version)
	}
}
