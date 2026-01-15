package rpm

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRPMSpec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/rpm/hello.spec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &rpmSpecParser{}
	deps, err := parser.Parse("hello.spec", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 7 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"gcc", "", core.Build},
		{"make", "", core.Build},
		{"gettext", ">= 0.19", core.Build},
		{"autoconf", "", core.Build},
		{"automake", "", core.Build},
		{"glibc", ">= 2.17", core.Runtime},
		{"info", "", core.Runtime},
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
}
