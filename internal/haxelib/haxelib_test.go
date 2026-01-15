package haxelib

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestHaxelibJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/haxelib/haxelib.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &haxelibJSONParser{}
	deps, err := parser.Parse("haxelib.json", content)
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

	if dep, ok := depMap["lime"]; !ok {
		t.Error("expected lime dependency")
	} else if dep.Version != "2.9.1" {
		t.Errorf("expected lime version 2.9.1, got %s", dep.Version)
	}

	if dep, ok := depMap["openfl"]; !ok {
		t.Error("expected openfl dependency")
	} else if dep.Version != "3.6.1" {
		t.Errorf("expected openfl version 3.6.1, got %s", dep.Version)
	}
}
