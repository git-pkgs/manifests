package pypi

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRequirementsTxt(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/requirements.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.txt", content)
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

	// Check Flask (version includes operator)
	if flask, ok := depMap["Flask"]; !ok {
		t.Error("expected Flask dependency")
	} else {
		if flask.Version != "== 0.8" {
			t.Errorf("Flask version = %q, want %q", flask.Version, "== 0.8")
		}
	}

	// Check package with dashes
	if sklearn, ok := depMap["scikit-learn"]; !ok {
		t.Error("expected scikit-learn dependency")
	} else {
		if sklearn.Version != "==0.16.1" {
			t.Errorf("scikit-learn version = %q, want %q", sklearn.Version, "==0.16.1")
		}
	}

	// Check >= constraint
	if beaker, ok := depMap["Beaker"]; !ok {
		t.Error("expected Beaker dependency")
	} else {
		if beaker.Version != ">=1.6.5" {
			t.Errorf("Beaker version = %q, want %q", beaker.Version, ">=1.6.5")
		}
	}

	// Verify comment was filtered out
	if _, ok := depMap["Jinja2"]; ok {
		t.Error("commented Jinja2 should not be included")
	}
}

func TestPipfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/Pipfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pipfileParser{}
	deps, err := parser.Parse("Pipfile", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestPipfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/Pipfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pipfileLockParser{}
	deps, err := parser.Parse("Pipfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check for integrity hashes
	hasIntegrity := false
	for _, d := range deps {
		if d.Integrity != "" {
			hasIntegrity = true
			break
		}
	}
	if !hasIntegrity {
		t.Error("expected at least one dependency with integrity hash")
	}
}

func TestPyprojectToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pyproject.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pyprojectParser{}
	deps, err := parser.Parse("pyproject.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestPoetryLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/poetry.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &poetryLockParser{}
	deps, err := parser.Parse("poetry.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check for version information
	hasVersion := false
	for _, d := range deps {
		if d.Version != "" {
			hasVersion = true
			break
		}
	}
	if !hasVersion {
		t.Error("expected at least one dependency with version")
	}
}

func TestRequirementsDevTxt(t *testing.T) {
	// Tests pip-compile output format (requirements-dev.txt)
	content, err := os.ReadFile("../../testdata/pypi/requirements-dev.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements-dev.txt", content)
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

	// Check pinned version from pip-compile
	if astroid, ok := depMap["astroid"]; !ok {
		t.Error("expected astroid dependency")
	} else if astroid.Version != "==2.9.0" {
		t.Errorf("astroid version = %q, want %q", astroid.Version, "==2.9.0")
	}

	// Check package with extras
	if cov, ok := depMap["coverage"]; !ok {
		t.Error("expected coverage dependency")
	} else if cov.Version != "==6.2" {
		t.Errorf("coverage version = %q, want %q", cov.Version, "==6.2")
	}
}

func TestRequirementsFrozen(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/requirements.frozen")
	if err != nil {
		t.Skipf("frozen fixture not found: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.frozen", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestPdmLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pdm.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pdmLockParser{}
	deps, err := parser.Parse("pdm.lock", content)
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

	// Check certifi
	if dep, ok := depMap["certifi"]; !ok {
		t.Error("expected certifi dependency")
	} else if dep.Version != "2024.2.2" {
		t.Errorf("certifi version = %q, want %q", dep.Version, "2024.2.2")
	}

	// Check requests
	if dep, ok := depMap["requests"]; !ok {
		t.Error("expected requests dependency")
	} else if dep.Version != "2.31.0" {
		t.Errorf("requests version = %q, want %q", dep.Version, "2.31.0")
	}

	// Check pytest is dev dependency
	if dep, ok := depMap["pytest"]; !ok {
		t.Error("expected pytest dependency")
	} else if dep.Scope != core.Development {
		t.Errorf("pytest scope = %v, want Development", dep.Scope)
	}
}

func TestUvLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/uv.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &uvLockParser{}
	deps, err := parser.Parse("uv.lock", content)
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

	// Check alabaster
	if dep, ok := depMap["alabaster"]; !ok {
		t.Error("expected alabaster dependency")
	} else if dep.Version != "0.7.16" {
		t.Errorf("alabaster version = %q, want %q", dep.Version, "0.7.16")
	}

	// Check babel
	if dep, ok := depMap["babel"]; !ok {
		t.Error("expected babel dependency")
	} else if dep.Version != "2.16.0" {
		t.Errorf("babel version = %q, want %q", dep.Version, "2.16.0")
	}

	// Check integrity is captured
	if dep, ok := depMap["certifi"]; !ok {
		t.Error("expected certifi dependency")
	} else if dep.Integrity == "" {
		t.Error("expected certifi to have integrity")
	}
}

func TestParsePEP508(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"requests>=2.0", "requests", ">=2.0"},
		{"requests[security]>=2.0", "requests", ">=2.0"},
		{"Django>=3.0,<4.0", "Django", ">=3.0,<4.0"},
		{"pytest", "pytest", ""},
		{"black==22.3.0", "black", "==22.3.0"},
		{"numpy>=1.20; python_version>='3.8'", "numpy", ">=1.20"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotName, gotVer := parsePEP508(tt.input)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotVer != tt.wantVersion {
				t.Errorf("version = %q, want %q", gotVer, tt.wantVersion)
			}
		})
	}
}
