package lake

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestLakefileToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/lake/lakefile.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &lakefileTomlParser{}
	deps, err := parser.Parse("lakefile.toml", content)
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

	if _, ok := depMap["localpkg"]; ok {
		t.Error("local path dependency should be skipped")
	}

	expected := map[string]string{
		"leanprover-community/batteries": "v4.30.0-rc2",
		"Cli":                            "main",
		"leanprover-community/mathlib":   "4.30.0",
		"leansqlite":                     "v0.1.0",
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
		if !dep.Direct {
			t.Errorf("%s should be direct", name)
		}
	}

	if depMap["Cli"].RegistryURL != "https://github.com/leanprover/lean4-cli" {
		t.Errorf("Cli RegistryURL = %q", depMap["Cli"].RegistryURL)
	}
	if depMap["leansqlite"].RegistryURL != "https://github.com/leanprover/leansqlite" {
		t.Errorf("leansqlite RegistryURL = %q", depMap["leansqlite"].RegistryURL)
	}
}

func TestLakefileLean(t *testing.T) {
	content, err := os.ReadFile("../../testdata/lake/lakefile.lean")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &lakefileLeanParser{}
	deps, err := parser.Parse("lakefile.lean", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d: %+v", len(deps), deps)
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	if _, ok := depMap["localpkg"]; ok {
		t.Error("local path dependency should be skipped")
	}
	if _, ok := depMap["disabled"]; ok {
		t.Error("commented-out dependency should be skipped")
	}

	expected := map[string]struct {
		version string
		url     string
	}{
		"leanprover-community/batteries": {"v4.30.0-rc2", ""},
		"leanprover-community/aesop":     {"4.30.0", ""},
		"MD4Lean":                        {"main", "https://github.com/acmepjz/md4lean"},
		"UnicodeBasic":                   {"v1.0.0", "https://github.com/fgdorais/lean4-unicode-basic"},
		"Cli":                            {"", "https://github.com/leanprover/lean4-cli"},
	}

	for name, want := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != want.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, want.version)
		}
		if dep.RegistryURL != want.url {
			t.Errorf("%s RegistryURL = %q, want %q", name, dep.RegistryURL, want.url)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %q, want runtime", name, dep.Scope)
		}
	}
}

func TestLakeManifestJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/lake/lake-manifest.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &lakeManifestParser{}
	deps, err := parser.Parse("lake-manifest.json", content)
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

	if _, ok := depMap["localpkg"]; ok {
		t.Error("local path dependency should be skipped")
	}

	batteries, ok := depMap["leanprover-community/batteries"]
	if !ok {
		t.Fatal("expected leanprover-community/batteries dependency")
	}
	if batteries.Version != "5c57f3857ba81924a88b2cdf4f062e34ec04ff11" {
		t.Errorf("batteries version = %q", batteries.Version)
	}
	if !batteries.Direct {
		t.Error("batteries should be direct (inherited=false)")
	}

	cli, ok := depMap["Cli"]
	if !ok {
		t.Fatal("expected Cli dependency")
	}
	if cli.RegistryURL != "https://github.com/leanprover/lean4-cli" {
		t.Errorf("Cli RegistryURL = %q", cli.RegistryURL)
	}

	plausible, ok := depMap["plausible"]
	if !ok {
		t.Fatal("expected plausible dependency")
	}
	if plausible.Direct {
		t.Error("plausible should be transitive (inherited=true)")
	}
}
