package arch

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPKGBUILD(t *testing.T) {
	content, err := os.ReadFile("../../testdata/arch/PKGBUILD")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pkgbuildParser{}
	deps, err := parser.Parse("PKGBUILD", content)
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

	// Check runtime dependency with version
	if glibc, ok := depMap["glibc"]; !ok {
		t.Error("expected glibc dependency")
	} else {
		if glibc.Version != ">=2.17" {
			t.Errorf("glibc version = %q, want %q", glibc.Version, ">=2.17")
		}
		if glibc.Scope != core.Runtime {
			t.Errorf("glibc scope = %q, want %q", glibc.Scope, core.Runtime)
		}
	}

	// Check runtime dependency without version
	if sh, ok := depMap["sh"]; !ok {
		t.Error("expected sh dependency")
	} else {
		if sh.Version != "" {
			t.Errorf("sh version = %q, want empty", sh.Version)
		}
		if sh.Scope != core.Runtime {
			t.Errorf("sh scope = %q, want %q", sh.Scope, core.Runtime)
		}
	}

	// Check build dependency
	if gcc, ok := depMap["gcc"]; !ok {
		t.Error("expected gcc dependency")
	} else {
		if gcc.Scope != core.Build {
			t.Errorf("gcc scope = %q, want %q", gcc.Scope, core.Build)
		}
	}

	// Check build dependency with version
	if gettext, ok := depMap["gettext"]; !ok {
		t.Error("expected gettext dependency")
	} else {
		if gettext.Version != ">=0.19" {
			t.Errorf("gettext version = %q, want %q", gettext.Version, ">=0.19")
		}
		if gettext.Scope != core.Build {
			t.Errorf("gettext scope = %q, want %q", gettext.Scope, core.Build)
		}
	}

	// Check test dependency
	if dejagnu, ok := depMap["dejagnu"]; !ok {
		t.Error("expected dejagnu dependency")
	} else {
		if dejagnu.Scope != core.Test {
			t.Errorf("dejagnu scope = %q, want %q", dejagnu.Scope, core.Test)
		}
	}

	if python, ok := depMap["python"]; !ok {
		t.Error("expected python dependency")
	} else {
		if python.Scope != core.Test {
			t.Errorf("python scope = %q, want %q", python.Scope, core.Test)
		}
	}
}
