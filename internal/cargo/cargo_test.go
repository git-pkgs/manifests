package cargo

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestCargoToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cargo/Cargo.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cargoTomlParser{}
	deps, err := parser.Parse("Cargo.toml", content)
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

	// Check rustc-serialize with wildcard version
	if rs, ok := depMap["rustc-serialize"]; !ok {
		t.Error("expected rustc-serialize dependency")
	} else {
		if rs.Version != "*" {
			t.Errorf("rustc-serialize version = %q, want %q", rs.Version, "*")
		}
		if rs.Scope != core.Runtime {
			t.Errorf("rustc-serialize scope = %q, want %q", rs.Scope, core.Runtime)
		}
	}

	// Check regex with table version
	if regex, ok := depMap["regex"]; !ok {
		t.Error("expected regex dependency")
	} else {
		if regex.Version != "*" {
			t.Errorf("regex version = %q, want %q", regex.Version, "*")
		}
	}

	// Check tempdir dev dependency
	if tempdir, ok := depMap["tempdir"]; !ok {
		t.Error("expected tempdir dependency")
	} else {
		if tempdir.Version != "0.3" {
			t.Errorf("tempdir version = %q, want %q", tempdir.Version, "0.3")
		}
		if tempdir.Scope != core.Development {
			t.Errorf("tempdir scope = %q, want %q", tempdir.Scope, core.Development)
		}
	}

	// Verify local_crate (path dependency) is filtered out
	if _, ok := depMap["local_crate"]; ok {
		t.Error("local_crate path dependency should be filtered out")
	}
}

func TestCargoLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cargo/Cargo.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cargoLockParser{}
	deps, err := parser.Parse("Cargo.lock", content)
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

	// Check regex with version and integrity
	if regex, ok := depMap["regex"]; !ok {
		t.Error("expected regex dependency")
	} else {
		if regex.Version != "1.6.0" {
			t.Errorf("regex version = %q, want %q", regex.Version, "1.6.0")
		}
		if regex.Integrity == "" {
			t.Error("expected regex to have integrity hash")
		}
		wantIntegrity := "sha256-4c4eb3267174b8c6c2f654116623910a0fef09c4753f8dd83db29c48a0df988b"
		if regex.Integrity != wantIntegrity {
			t.Errorf("regex integrity = %q, want %q", regex.Integrity, wantIntegrity)
		}
	}

	// Verify local_crate (no source) is filtered out
	if _, ok := depMap["local_crate"]; ok {
		t.Error("local_crate should be filtered out (no source)")
	}

	// Verify update (root package, no source) is filtered out
	if _, ok := depMap["update"]; ok {
		t.Error("update root package should be filtered out")
	}
}
