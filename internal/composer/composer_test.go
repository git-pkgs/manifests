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

	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 4 packages with exact versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"laravel/framework": {"5.0.*", core.Runtime},
		"drupal/address":    {"^1.0", core.Runtime},
		"phpunit/phpunit":   {"~4.0", core.Development},
		"phpspec/phpspec":   {"~2.1", core.Development},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
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

	if len(deps) != 10 {
		t.Fatalf("expected 10 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 10 packages with versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"doctrine/annotations":      {"v1.2.1", core.Runtime},
		"doctrine/cache":            {"v1.3.1", core.Runtime},
		"doctrine/collections":      {"v1.2", core.Runtime},
		"drupal/address":            {"1.9.0", core.Runtime},
		"symfony/monolog-bundle":    {"v2.6.1", core.Runtime},
		"symfony/swiftmailer-bundle": {"v2.3.8", core.Runtime},
		"symfony/symfony":           {"v2.6.1", core.Runtime},
		"twig/extensions":           {"v1.2.0", core.Runtime},
		"twig/twig":                 {"v1.16.2", core.Runtime},
		"sensio/generator-bundle":   {"v2.5.0", core.Development},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
		}
	}
}
