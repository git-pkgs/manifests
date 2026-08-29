package gleam

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGleamToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gleam/gleam.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gleamTomlParser{}
	res, err := parser.Parse("gleam.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All 3 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"gleam_stdlib", ">= 0.53.0 and < 2.0.0", core.Runtime},
		{"gleam_http", "~> 3.0", core.Runtime},
		{"gleeunit", ">= 1.3.0 and < 2.0.0", core.Development},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}

	wantDeclarations := map[string]core.Scope{
		"dependencies/gleam_stdlib": core.Runtime,
		"dependencies/gleam_http":   core.Runtime,
		"dev-dependencies/gleeunit": core.Development,
	}
	if len(res.Declarations) != len(wantDeclarations) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(res.Declarations), len(wantDeclarations), res.Declarations)
	}
	for _, declaration := range res.Declarations {
		if scope, ok := wantDeclarations[declaration.Location]; !ok || declaration.Scope != scope || !declaration.Direct {
			t.Errorf("unexpected declaration: %+v", declaration)
		}
	}
}

func TestGleamTomlDeclarationsIgnoreNonHexSources(t *testing.T) {
	content := []byte(`name = "example"
version = "1.0.0"

[dependencies]
gleam_stdlib = ">= 1.0.0 and < 2.0.0"
local_package = { path = "../local_package" }
git_package = { git = "https://example.com/git_package.git", ref = "main" }

[dev-dependencies]
gleeunit = ">= 1.0.0 and < 2.0.0"
`)

	result, err := (&gleamTomlParser{}).Parse("gleam.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	want := map[string]core.Scope{
		"dependencies/gleam_stdlib": core.Runtime,
		"dev-dependencies/gleeunit": core.Development,
	}
	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		if scope, ok := want[declaration.Location]; !ok || declaration.Scope != scope || !declaration.Direct {
			t.Errorf("unexpected declaration: %+v", declaration)
		}
	}
	if len(result.Dependencies) != len(want) {
		t.Fatalf("Dependencies has %d entries, want %d: %+v", len(result.Dependencies), len(want), result.Dependencies)
	}
}
