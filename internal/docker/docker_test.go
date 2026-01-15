package docker

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestDockerfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/docker/Dockerfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &dockerfileParser{}
	deps, err := parser.Parse("Dockerfile", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// The 1 package with version and scope
	dep, ok := depMap["ruby"]
	if !ok {
		t.Fatal("expected ruby dependency")
	}
	if dep.Version != "3.1.2-alpine" {
		t.Errorf("ruby version = %q, want %q", dep.Version, "3.1.2-alpine")
	}
	if dep.Scope != core.Runtime {
		t.Errorf("ruby scope = %v, want %v", dep.Scope, core.Runtime)
	}
}

func TestDockerCompose(t *testing.T) {
	content, err := os.ReadFile("../../testdata/docker/docker-compose.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &dockerComposeParser{}
	deps, err := parser.Parse("docker-compose.yml", content)
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

	// All 2 packages with versions
	expected := map[string]string{
		"postgres": "9.6-alpine",
		"redis":    "4.0-alpine",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, core.Runtime)
		}
	}
}
