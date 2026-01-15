package clojure

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestProjectClj(t *testing.T) {
	content, err := os.ReadFile("../../testdata/clojure/project.clj")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &projectCljParser{}
	deps, err := parser.Parse("project.clj", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 6 {
		t.Fatalf("expected 6 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 6 packages with versions
	expected := map[string]string{
		"org.clojure/clojure":    "1.6.0",
		"cheshire":               "5.4.0",
		"compojure":              "1.3.2",
		"ring/ring-defaults":     "0.1.2",
		"ring/ring-jetty-adapter": "1.2.1",
		"lein-ring":              "0.8.13",
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
