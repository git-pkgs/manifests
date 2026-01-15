package luarocks

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRockspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/luarocks/example.rockspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &rockspecParser{}
	deps, err := parser.Parse("example.rockspec", content)
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

	// Check lua with >= constraint
	if dep, ok := depMap["lua"]; !ok {
		t.Error("expected lua dependency")
	} else if dep.Version != ">= 5.1" {
		t.Errorf("expected lua version >= 5.1, got %s", dep.Version)
	}

	// Check luafilesystem with >= constraint
	if dep, ok := depMap["luafilesystem"]; !ok {
		t.Error("expected luafilesystem dependency")
	} else if dep.Version != ">= 1.8.0" {
		t.Errorf("expected luafilesystem version >= 1.8.0, got %s", dep.Version)
	}

	// Check lpeg with ~> constraint
	if dep, ok := depMap["lpeg"]; !ok {
		t.Error("expected lpeg dependency")
	} else if dep.Version != "~> 1.0" {
		t.Errorf("expected lpeg version ~> 1.0, got %s", dep.Version)
	}

	// Check luasocket without version
	if dep, ok := depMap["luasocket"]; !ok {
		t.Error("expected luasocket dependency")
	} else if dep.Version != "" {
		t.Errorf("expected luasocket version empty, got %s", dep.Version)
	}
}
