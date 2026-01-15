package composer

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestComposerJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/composer/composer.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &composerJSONParser{}
	deps, err := parser.Parse("composer.json", content)
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

	// Check runtime dependency
	if laravel, ok := depMap["laravel/framework"]; !ok {
		t.Error("expected laravel/framework dependency")
	} else {
		if laravel.Version != "5.0.*" {
			t.Errorf("laravel/framework version = %q, want %q", laravel.Version, "5.0.*")
		}
		if laravel.Scope != core.Runtime {
			t.Errorf("laravel/framework scope = %q, want %q", laravel.Scope, core.Runtime)
		}
	}

	// Check dev dependency
	if phpunit, ok := depMap["phpunit/phpunit"]; !ok {
		t.Error("expected phpunit/phpunit dependency")
	} else {
		if phpunit.Scope != core.Development {
			t.Errorf("phpunit/phpunit scope = %q, want %q", phpunit.Scope, core.Development)
		}
	}
}

func TestComposerLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/composer/composer.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &composerLockParser{}
	deps, err := parser.Parse("composer.lock", content)
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
