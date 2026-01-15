package conan

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestConanfileTxt(t *testing.T) {
	content, err := os.ReadFile("../../testdata/conan/conanfile.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &conanfileTxtParser{}
	deps, err := parser.Parse("conanfile.txt", content)
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

	// Check runtime dependencies
	if dep, ok := depMap["zlib"]; !ok {
		t.Error("expected zlib dependency")
	} else {
		if dep.Version != "1.2.11" {
			t.Errorf("expected zlib version 1.2.11, got %s", dep.Version)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("expected zlib scope Runtime, got %v", dep.Scope)
		}
	}

	if dep, ok := depMap["boost"]; !ok {
		t.Error("expected boost dependency")
	} else {
		if dep.Version != "1.76.0" {
			t.Errorf("expected boost version 1.76.0, got %s", dep.Version)
		}
	}

	// Check build dependency
	if dep, ok := depMap["cmake"]; !ok {
		t.Error("expected cmake dependency")
	} else {
		if dep.Version != "3.21.0" {
			t.Errorf("expected cmake version 3.21.0, got %s", dep.Version)
		}
		if dep.Scope != core.Build {
			t.Errorf("expected cmake scope Build, got %v", dep.Scope)
		}
	}
}

func TestConanfilePy(t *testing.T) {
	content, err := os.ReadFile("../../testdata/conan/conanfile.py")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &conanfilePyParser{}
	deps, err := parser.Parse("conanfile.py", content)
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

	// Check zlib
	if dep, ok := depMap["zlib"]; !ok {
		t.Error("expected zlib dependency")
	} else if dep.Version != "1.2.11" {
		t.Errorf("expected zlib version 1.2.11, got %s", dep.Version)
	}

	// Check boost (with @user/channel suffix stripped)
	if dep, ok := depMap["boost"]; !ok {
		t.Error("expected boost dependency")
	} else if dep.Version != "1.76.0" {
		t.Errorf("expected boost version 1.76.0, got %s", dep.Version)
	}
}

func TestConanLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/conan/conan.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &conanLockParser{}
	deps, err := parser.Parse("conan.lock", content)
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
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"zlib", "1.2.11", core.Runtime},
		{"boost", "1.76.0", core.Runtime},
		{"cmake", "3.21.0", core.Build},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}
