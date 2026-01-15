package rpm

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRPMSpec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/rpm/hello.spec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &rpmSpecParser{}
	deps, err := parser.Parse("hello.spec", content)
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

	// Check build dependency
	if dep, ok := depMap["gcc"]; !ok {
		t.Error("expected gcc build dependency")
	} else if dep.Scope != core.Build {
		t.Errorf("expected gcc scope Build, got %v", dep.Scope)
	}

	// Check build dependency with version
	if dep, ok := depMap["gettext"]; !ok {
		t.Error("expected gettext dependency")
	} else {
		if dep.Version != ">= 0.19" {
			t.Errorf("expected gettext version >= 0.19, got %s", dep.Version)
		}
		if dep.Scope != core.Build {
			t.Errorf("expected gettext scope Build, got %v", dep.Scope)
		}
	}

	// Check runtime dependency
	if dep, ok := depMap["glibc"]; !ok {
		t.Error("expected glibc runtime dependency")
	} else {
		if dep.Scope != core.Runtime {
			t.Errorf("expected glibc scope Runtime, got %v", dep.Scope)
		}
		if dep.Version != ">= 2.17" {
			t.Errorf("expected glibc version >= 2.17, got %s", dep.Version)
		}
	}

	// Check autoconf, automake (comma-separated)
	if _, ok := depMap["autoconf"]; !ok {
		t.Error("expected autoconf dependency")
	}
	if _, ok := depMap["automake"]; !ok {
		t.Error("expected automake dependency")
	}
}
