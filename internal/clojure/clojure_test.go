package clojure

import (
	"os"
	"testing"
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check that dependencies have names
	for _, d := range deps {
		if d.Name == "" {
			t.Error("expected all dependencies to have names")
		}
	}
}
