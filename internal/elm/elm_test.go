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

	// All 4 packages with versions
	expected := map[string]string{
		"elm-lang/core":         "1.0.0 <= v < 2.0.0",
		"evancz/elm-markdown":   "1.1.0 <= v < 2.0.0",
		"evancz/elm-html":       "1.0.0 <= v < 2.0.0",
		"evancz/local-channel":  "1.0.0 <= v < 2.0.0",
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
		if !dep.Direct {
			t.Errorf("%s should be direct dependency", name)
		}
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

	// All 2 packages with versions
	expected := map[string]string{
		"johnpmayer/elm-webgl":         "0.1.1",
		"johnpmayer/elm-linear-algebra": "1.0.1",
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
