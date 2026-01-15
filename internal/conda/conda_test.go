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
