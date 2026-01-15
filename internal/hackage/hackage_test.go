package hackage

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestCabal(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/example.cabal")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cabalParser{}
	deps, err := parser.Parse("example.cabal", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check dependency with version constraint
	if dep, ok := depMap["aeson"]; !ok {
		t.Error("expected aeson dependency")
	} else if dep.Version != "== 1.1.*" {
		t.Errorf("expected aeson version == 1.1.*, got %s", dep.Version)
	}

	// Check dependency with range constraint
	if dep, ok := depMap["base"]; !ok {
		t.Error("expected base dependency")
	} else if dep.Version != ">= 4.9 && < 4.11" {
		t.Errorf("expected base version >= 4.9 && < 4.11, got %s", dep.Version)
	}
}

func TestStackLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/stack.yaml.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &stackLockParser{}
	deps, err := parser.Parse("stack.yaml.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check aeson
	if dep, ok := depMap["aeson"]; !ok {
		t.Error("expected aeson dependency")
	} else if dep.Version != "2.1.2.1" {
		t.Errorf("expected aeson version 2.1.2.1, got %s", dep.Version)
	}

	// Check text
	if dep, ok := depMap["text"]; !ok {
		t.Error("expected text dependency")
	} else if dep.Version != "2.0.2" {
		t.Errorf("expected text version 2.0.2, got %s", dep.Version)
	}
}

func TestCabalConfig(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/cabal.config")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cabalConfigParser{}
	deps, err := parser.Parse("cabal.config", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check a constraint
	if dep, ok := depMap["aeson"]; !ok {
		t.Error("expected aeson dependency")
	} else if dep.Version != "1.1.2.0" {
		t.Errorf("expected aeson version 1.1.2.0, got %s", dep.Version)
	}

	// Check bytestring
	if dep, ok := depMap["bytestring"]; !ok {
		t.Error("expected bytestring dependency")
	} else if dep.Version != "0.10.8.2" {
		t.Errorf("expected bytestring version 0.10.8.2, got %s", dep.Version)
	}
}

func TestCabalProjectFreeze(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/cabal.project.freeze")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cabalFreezeParser{}
	deps, err := parser.Parse("cabal.project.freeze", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}
