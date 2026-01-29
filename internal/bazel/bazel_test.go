package bazel

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestBazelModuleManifest(t *testing.T) {
	content, err := os.ReadFile("../../testdata/bazel/MODULE.bazel")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &bazelModuleManifestParser{}
	deps, err := parser.Parse("MODULE.bazel", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 9 {
		t.Fatalf("expected 9 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"google_benchmark":    {"1.9.4", core.Development},
		"j2cl":                {"20250630", core.Build},
		"jsinterop_generator": {"20250812", core.Build},
		"jsinterop_base":      {"1.1.0", core.Build},
		"bazel_skylib":        {"1.7.1", core.Build},
		"google_bazel_common": {"0.0.1", core.Build},
		"rules_java":          {"8.13.0", core.Build},
		"rules_license":       {"1.0.0", core.Build},
		"rules_jvm_external":  {"6.6", core.Build},
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

func TestBazelModuleManifest_Parse_Error(t *testing.T) {
	content, err := os.ReadFile("../../testdata/bazel/MODULE2.bazel")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &bazelModuleManifestParser{}
	result, err := parser.Parse("MODULE.bazel", content)

	if err == nil || result != nil {
		t.Fatalf("expected error, got nil.")
	}

	if err.Error() != `failed to parse MODULE.bazel: 5:bazel_dep 'name' "" has invalid format` {
		t.Fatalf("unexpected error: %v", err)
	}
}
