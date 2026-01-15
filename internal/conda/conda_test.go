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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check dependency with version=build format
	if dep, ok := depMap["beautifulsoup4"]; !ok {
		t.Error("expected beautifulsoup4 dependency")
	} else if dep.Version != "4.7.1" {
		t.Errorf("expected beautifulsoup4 version 4.7.1, got %s", dep.Version)
	}

	// Check dependency with just name
	if _, ok := depMap["pip"]; !ok {
		t.Error("expected pip dependency")
	}

	// Check dependency with version
	if dep, ok := depMap["numpy"]; !ok {
		t.Error("expected numpy dependency")
	} else if dep.Version != "1.16.4" {
		t.Errorf("expected numpy version 1.16.4, got %s", dep.Version)
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

	// Should only include conda dependencies, not pip dependencies
	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// pip and sqlite are conda deps
	if _, ok := depMap["pip"]; !ok {
		t.Error("expected pip dependency")
	}
	if _, ok := depMap["sqlite"]; !ok {
		t.Error("expected sqlite dependency")
	}

	// Django and urllib3 are pip deps, should not be included
	if _, ok := depMap["Django"]; ok {
		t.Error("expected Django (pip dep) to be excluded")
	}
	if _, ok := depMap["urllib3"]; ok {
		t.Error("expected urllib3 (pip dep) to be excluded")
	}
}
