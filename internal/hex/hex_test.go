package hex

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestMixExs(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hex/mix.exs")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &mixExsParser{}
	deps, err := parser.Parse("mix.exs", content)
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
		"poison": "~> 1.3.1",
		"plug":   "~> 0.11.0",
		"cowboy": "~> 1.0.0",
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

func TestMixLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hex/mix.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &mixLockParser{}
	deps, err := parser.Parse("mix.lock", content)
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

	// All 5 packages with versions and integrities
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"cowboy": {"1.0.4", "sha256-a324a8df9f2316c833a470d918aaf73ae894278b8aa6226ce7a9bf699388f878"},
		"cowlib": {"1.0.2", "sha256-9d769a1d062c9c3ac753096f868ca121e2730b9a377de23dec0f7e08b1df84ee"},
		"plug":   {"1.1.6", "sha256-8927e4028433fcb859e000b9389ee9c37c80eb28378eeeea31b0273350bf668b"},
		"poison": {"2.1.0", "sha256-f583218ced822675e484648fa26c933d621373f01c6c76bd00005d7bd4b82e27"},
		"ranch":  {"1.2.1", "sha256-a6fb992c10f2187b46ffd17ce398ddf8a54f691b81768f9ef5f461ea7e28c762"},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, exp.integrity)
		}
	}
}
