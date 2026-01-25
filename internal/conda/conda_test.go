package conda

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestCondaEnvironment(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/environment.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &condaEnvParser{}
	deps, err := parser.Parse("environment.yml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 11 {
		t.Fatalf("expected 11 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 11 packages with versions
	expected := map[string]string{
		"beautifulsoup4": "4.7.1",
		"biopython":      "1.74",
		"certifi":        "2019.6.16",
		"ncurses":        "6.1",
		"numpy":          "1.16.4",
		"openssl":        "1.1.1c",
		"pip":            "",
		"python":         "3.7.3",
		"readline":       "7.0",
		"setuptools":     "",
		"sqlite":         "3.29.0",
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

func TestCondaEnvironmentWithPip(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/conda_with_pip/environment.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &condaEnvParser{}
	deps, err := parser.Parse("environment.yml", content)
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

	// All 2 conda dependencies
	expected := map[string]string{
		"pip":    "",
		"sqlite": "3.29.0",
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

	// Django and urllib3 are pip deps, should not be included
	if _, ok := depMap["Django"]; ok {
		t.Error("expected Django (pip dep) to be excluded")
	}
	if _, ok := depMap["urllib3"]; ok {
		t.Error("expected urllib3 (pip dep) to be excluded")
	}
}

func TestCondaLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/conda/conda-lock.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &condaLockParser{}
	deps, err := parser.Parse("conda-lock.yml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// 4 conda packages (pip packages are excluded)
	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Verify conda packages
	expected := []struct {
		name        string
		version     string
		scope       core.Scope
		hasIntegrity bool
	}{
		{"python", "3.11.0", core.Runtime, true},
		{"numpy", "1.24.3", core.Runtime, true},
		{"pandas", "2.0.1", core.Runtime, true},
		{"pytest", "7.3.1", core.Development, true},
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
		if exp.hasIntegrity && dep.Integrity == "" {
			t.Errorf("%s should have integrity hash", exp.name)
		}
	}

	// pip packages should be excluded
	if _, ok := depMap["requests"]; ok {
		t.Error("expected requests (pip package) to be excluded")
	}
	if _, ok := depMap["black"]; ok {
		t.Error("expected black (pip package) to be excluded")
	}
}
