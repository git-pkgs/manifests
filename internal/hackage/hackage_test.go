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

	if len(deps) != 19 {
		t.Fatalf("expected 19 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions (parser also picks up extensions)
	expected := map[string]string{
		"aeson":          "== 1.1.*",
		"base":           ">= 4.9 && < 4.11",
		"Cabal":          "== 2.0.*",
		"envy":           "== 1.3.*",
		"servant-server": "== 0.11.*",
		"text":           "== 1.2.*",
		"warp":           "== 3.2.*",
		"hspec":          "== 2.4.*",
		"bytestring":     "== 0.10.*",
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

	// All 3 packages with versions
	expected := map[string]string{
		"aeson":      "2.1.2.1",
		"text":       "2.0.2",
		"bytestring": "0.11.5.3",
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

	if len(deps) != 108 {
		t.Fatalf("expected 108 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"aeson":      "1.1.2.0",
		"bytestring": "0.10.8.2",
		"base":       "4.10.1.0",
		"text":       "1.2.3.0",
		"warp":       "3.2.13",
		"Cabal":      "2.0.1.0",
	}

	for name, wantVer := range samples {
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

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 5 packages with versions
	expected := map[string]string{
		"Cabal":      "3.12.0.0",
		"aeson":      "2.2.3.0",
		"base":       "4.18.2.1",
		"bytestring": "0.11.5.3",
		"text":       "2.0.2",
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
