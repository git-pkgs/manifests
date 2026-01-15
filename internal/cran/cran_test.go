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

	if len(deps) != 26 {
		t.Fatalf("expected 26 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 26 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"digest", "", core.Runtime},
		{"grid", "", core.Runtime},
		{"gtable", ">= 0.1.1", core.Runtime},
		{"MASS", "", core.Runtime},
		{"plyr", ">= 1.7.1", core.Runtime},
		{"reshape2", "", core.Runtime},
		{"scales", ">= 0.3.0", core.Runtime},
		{"stats", "", core.Runtime},
		{"covr", "", core.Development},
		{"ggplot2movies", "", core.Development},
		{"hexbin", "", core.Development},
		{"Hmisc", "", core.Development},
		{"lattice", "", core.Development},
		{"mapproj", "", core.Development},
		{"maps", "", core.Development},
		{"maptools", "", core.Development},
		{"mgcv", "", core.Development},
		{"multcomp", "", core.Development},
		{"nlme", "", core.Development},
		{"testthat", ">= 0.11.0", core.Development},
		{"quantreg", "", core.Development},
		{"knitr", "", core.Development},
		{"rpart", "", core.Development},
		{"rmarkdown", "", core.Development},
		{"svglite", "", core.Development},
		{"sp", "", core.Optional},
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

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 5 packages with versions and integrities
	expected := []struct {
		name      string
		version   string
		integrity string
	}{
		{"ggplot2", "3.4.4", "md5-e3c4d693c5c82602ea6c9ff5583c32ad"},
		{"tidyr", "1.3.0", "md5-8fdd7c61b8f83a8b1a5a1cec8e1d7dcc"},
		{"rmarkdown", "2.25", "md5-88a70f9d3e5e5b3e5e5e5e5e5e5e5e5e"},
		{"localpackage", "0.1.0", ""},
		{"dplyr", "1.1.4", "md5-597b74c671d8bffb59c3aa51e8f7db53"},
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
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", exp.name, dep.Integrity, exp.integrity)
		}
	}
}

func TestDescription2(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cran/DESCRIPTION2")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &descriptionParser{}
	deps, err := parser.Parse("DESCRIPTION", content)
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

	// All 19 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"methods", "", core.Runtime},
		{"chron", "", core.Runtime},
		{"ggplot2", ">= 0.9.0", core.Development},
		{"plyr", "", core.Development},
		{"reshape", "", core.Development},
		{"reshape2", "", core.Development},
		{"testthat", ">= 0.4", core.Development},
		{"hexbin", "", core.Development},
		{"fastmatch", "", core.Development},
		{"nlme", "", core.Development},
		{"xts", "", core.Development},
		{"bit64", "", core.Development},
		{"gdata", "", core.Development},
		{"GenomicRanges", "", core.Development},
		{"caret", "", core.Development},
		{"knitr", "", core.Development},
		{"curl", "", core.Development},
		{"zoo", "", core.Development},
		{"plm", "", core.Development},
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
