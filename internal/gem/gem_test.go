package gem

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGemfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/Gemfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileParser{}
	deps, err := parser.Parse("Gemfile", content)
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

	// Check rails dependency with exact version
	if rails, ok := depMap["rails"]; !ok {
		t.Error("expected rails dependency")
	} else {
		if rails.Version != "4.2.0" {
			t.Errorf("rails version = %q, want %q", rails.Version, "4.2.0")
		}
		if rails.Scope != core.Runtime {
			t.Errorf("rails scope = %q, want %q", rails.Scope, core.Runtime)
		}
	}

	// Check nokogiri with pessimistic version
	if nokogiri, ok := depMap["nokogiri"]; !ok {
		t.Error("expected nokogiri dependency")
	} else {
		if nokogiri.Version != "~> 1.6" {
			t.Errorf("nokogiri version = %q, want %q", nokogiri.Version, "~> 1.6")
		}
	}

	// Check spring is in development scope
	if spring, ok := depMap["spring"]; !ok {
		t.Error("expected spring dependency")
	} else {
		if spring.Scope != core.Development {
			t.Errorf("spring scope = %q, want %q", spring.Scope, core.Development)
		}
	}

	// Check puma is in production (runtime) scope
	if puma, ok := depMap["puma"]; !ok {
		t.Error("expected puma dependency")
	} else {
		if puma.Scope != core.Runtime {
			t.Errorf("puma scope = %q, want %q", puma.Scope, core.Runtime)
		}
	}
}

func TestGemfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/Gemfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
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

	// Check actionmailer with version
	if am, ok := depMap["actionmailer"]; !ok {
		t.Error("expected actionmailer dependency")
	} else {
		if am.Version != "4.2.3" {
			t.Errorf("actionmailer version = %q, want %q", am.Version, "4.2.3")
		}
	}

	// Check direct dependency status
	if rails, ok := depMap["rails"]; ok {
		if !rails.Direct {
			t.Error("rails should be direct dependency")
		}
	}

	// Check transitive dependency status
	if hashie, ok := depMap["hashie"]; ok {
		if hashie.Direct {
			t.Error("hashie should be transitive dependency")
		}
	}
}

func TestGemfileLockWithChecksums(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/Gemfile-with-checksums.lock")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check rake has integrity hash
	if rake, ok := depMap["rake"]; !ok {
		t.Error("expected rake dependency")
	} else {
		if rake.Integrity == "" {
			t.Error("expected rake to have integrity hash")
		}
		if rake.Integrity != "sha256-46cb38dae65d7d74b6020a4ac9d48afed8eb8149c040eccf0523bec91907059d" {
			t.Errorf("rake integrity = %q", rake.Integrity)
		}
	}
}

func TestGemspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/devise.gemspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemspecParser{}
	deps, err := parser.Parse("devise.gemspec", content)
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

	// Check warden runtime dependency
	if warden, ok := depMap["warden"]; !ok {
		t.Error("expected warden dependency")
	} else {
		if warden.Version != "~> 1.2.3" {
			t.Errorf("warden version = %q, want %q", warden.Version, "~> 1.2.3")
		}
		if warden.Scope != core.Runtime {
			t.Errorf("warden scope = %q, want %q", warden.Scope, core.Runtime)
		}
	}

	// Check orm_adapter development dependency
	if orm, ok := depMap["orm_adapter"]; !ok {
		t.Error("expected orm_adapter dependency")
	} else {
		if orm.Scope != core.Development {
			t.Errorf("orm_adapter scope = %q, want %q", orm.Scope, core.Development)
		}
	}
}
