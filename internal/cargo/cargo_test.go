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

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 3 packages with exact versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"rustc-serialize": {"*", core.Runtime},
		"regex":           {"*", core.Runtime},
		"tempdir":         {"0.3", core.Development},
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

	if len(deps) != 16 {
		t.Fatalf("expected 16 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 16 packages with versions and integrities
	// Note: rand_core appears twice (0.3.1 and 0.4.2) so we check it separately
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"aho-corasick":                 {"0.7.18", "sha256-1e37cfd5e7657ada45f742d6e99ca5788580b5c529dc78faf11ece6dc702656f"},
		"fuchsia-cprng":                {"0.1.1", "sha256-a06f77d526c1a601b7c4cdd98f54b5eaabffc14d5f2f0296febdc7f357c6d3ba"},
		"libc":                         {"0.2.126", "sha256-349d5a591cd28b49e1d1037471617a32ddcda5731b99419008085f72d5a53836"},
		"memchr":                       {"2.5.0", "sha256-2dffe52ecf27772e601905b7522cb4ef790d2cc203488bbd0e2fe85fcb74566d"},
		"rand":                         {"0.4.6", "sha256-552840b97013b1a26992c11eac34bdd778e464601a4c2054b5f0bff7c6761293"},
		"rdrand":                       {"0.4.0", "sha256-678054eb77286b51581ba43620cc911abf02758c91f93f479767aed0f90458b2"},
		"regex":                        {"1.6.0", "sha256-4c4eb3267174b8c6c2f654116623910a0fef09c4753f8dd83db29c48a0df988b"},
		"regex-syntax":                 {"0.6.27", "sha256-a3f87b73ce11b1619a3c6332f45341e0047173771e8b8b73f87bfeefb7b56244"},
		"remove_dir_all":               {"0.5.3", "sha256-3acd125665422973a33ac9d3dd2df85edad0f4ae9b00dafb1a05e43a9f5ef8e7"},
		"rustc-serialize":              {"0.3.24", "sha256-dcf128d1287d2ea9d80910b5f1120d0b8eede3fbf1abe91c40d39ea7d51e6fda"},
		"tempdir":                      {"0.3.7", "sha256-15f2b5fb00ccdf689e0149d1b1b3c03fead81c2b37735d812fa8bddbbf41b6d8"},
		"winapi":                       {"0.3.9", "sha256-5c839a674fcd7a98952e593242ea400abe93992746761e38641405d28b00f419"},
		"winapi-i686-pc-windows-gnu":   {"0.4.0", "sha256-ac3b87c63620426dd9b991e5ce0329eff545bccbbb34f3be09ff6fb6ab51b7b6"},
		"winapi-x86_64-pc-windows-gnu": {"0.4.0", "sha256-712e227841d057c1ee1cd2fb22fa7e5a5461ae8e48fa2ca79ec42cfc1931183f"},
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
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, exp.integrity)
		}
	}

	// rand_core appears twice with different versions (0.3.1 and 0.4.2)
	randCoreVersions := map[string]string{
		"0.3.1": "sha256-7a6fdeb83b075e8266dcc8762c22776f6877a63111121f5f8c7411e5be7eed4b",
		"0.4.2": "sha256-9c33a3c44ca05fa6f1807d8e6743f3824e8509beca625669633be0acbdf509dc",
	}
	foundVersions := make(map[string]bool)
	for _, d := range deps {
		if d.Name == "rand_core" {
			foundVersions[d.Version] = true
			wantIntegrity, ok := randCoreVersions[d.Version]
			if !ok {
				t.Errorf("unexpected rand_core version %q", d.Version)
				continue
			}
			if d.Integrity != wantIntegrity {
				t.Errorf("rand_core %s integrity = %q, want %q", d.Version, d.Integrity, wantIntegrity)
			}
		}
	}
	for ver := range randCoreVersions {
		if !foundVersions[ver] {
			t.Errorf("expected rand_core %s dependency", ver)
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
