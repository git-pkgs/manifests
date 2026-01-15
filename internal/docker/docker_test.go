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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check ruby base image
	found := false
	for _, d := range deps {
		if d.Name == "ruby" && d.Version == "3.1.2-alpine" {
			found = true
			if d.Scope != core.Runtime {
				t.Errorf("ruby scope = %q, want %q", d.Scope, core.Runtime)
			}
			break
		}
	}
	if !found {
		t.Error("expected ruby:3.1.2-alpine dependency")
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	// Check that we have at least one image dependency
	hasImage := false
	for _, d := range deps {
		if d.Name != "" {
			hasImage = true
			break
		}
	}
	if !hasImage {
		t.Error("expected at least one image dependency")
	}
}
