package cran

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestDescription(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cran/DESCRIPTION")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &descriptionParser{}
	deps, err := parser.Parse("DESCRIPTION", content)
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

	// Check runtime dependency from Imports with version constraint
	if dep, ok := depMap["gtable"]; !ok {
		t.Error("expected gtable dependency")
	} else {
		if dep.Scope != core.Runtime {
			t.Errorf("expected gtable scope Runtime, got %v", dep.Scope)
		}
		if dep.Version != ">= 0.1.1" {
			t.Errorf("expected gtable version >= 0.1.1, got %s", dep.Version)
		}
	}

	// Check runtime dependency from Imports without version
	if dep, ok := depMap["digest"]; !ok {
		t.Error("expected digest dependency")
	} else if dep.Scope != core.Runtime {
		t.Errorf("expected digest scope Runtime, got %v", dep.Scope)
	}

	// Check development dependency from Suggests
	if dep, ok := depMap["testthat"]; !ok {
		t.Error("expected testthat dependency")
	} else {
		if dep.Scope != core.Development {
			t.Errorf("expected testthat scope Development, got %v", dep.Scope)
		}
		if dep.Version != ">= 0.11.0" {
			t.Errorf("expected testthat version >= 0.11.0, got %s", dep.Version)
		}
	}

	// Check optional dependency from Enhances
	if dep, ok := depMap["sp"]; !ok {
		t.Error("expected sp dependency")
	} else if dep.Scope != core.Optional {
		t.Errorf("expected sp scope Optional, got %v", dep.Scope)
	}

	// R itself should be excluded
	if _, ok := depMap["R"]; ok {
		t.Error("expected R dependency to be excluded")
	}
}

func TestRenvLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cran/renv.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &renvLockParser{}
	deps, err := parser.Parse("renv.lock", content)
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

	// Check dplyr
	if dep, ok := depMap["dplyr"]; !ok {
		t.Error("expected dplyr dependency")
	} else {
		if dep.Version != "1.1.4" {
			t.Errorf("expected dplyr version 1.1.4, got %s", dep.Version)
		}
		if dep.Integrity != "md5-597b74c671d8bffb59c3aa51e8f7db53" {
			t.Errorf("expected dplyr integrity md5-597b74c671d8bffb59c3aa51e8f7db53, got %s", dep.Integrity)
		}
	}

	// Check ggplot2
	if dep, ok := depMap["ggplot2"]; !ok {
		t.Error("expected ggplot2 dependency")
	} else if dep.Version != "3.4.4" {
		t.Errorf("expected ggplot2 version 3.4.4, got %s", dep.Version)
	}

	// Check local package (no hash means no integrity)
	if dep, ok := depMap["localpackage"]; !ok {
		t.Error("expected localpackage dependency")
	} else if dep.Integrity != "" {
		t.Errorf("expected localpackage to have no integrity, got %s", dep.Integrity)
	}
}
