package pub

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPubspecYAML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pub/pubspec.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pubspecYAMLParser{}
	res, err := parser.Parse("pubspec.yaml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All 4 packages with versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"analyzer":          {">=0.22.0 <0.25.0", core.Runtime},
		"args":              {">=0.12.0 <0.13.0", core.Runtime},
		"benchmark_harness": {">=1.0.0 <2.0.0", core.Development},
		"guinness":          {">=0.1.9 <0.2.0", core.Development},
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

func TestPubspecLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pub/pubspec.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pubspecLockParser{}
	res, err := parser.Parse("pubspec.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	expected := map[string]struct {
		version   string
		integrity string
		scope     core.Scope
		direct    bool
	}{
		"collection": {
			"1.19.1",
			"sha256-2f5709ae4d3d59dd8f7cd309b4e023046b57d8a6c82130785d2b0e5868084e76",
			core.Runtime,
			false,
		},
		"lints": {
			"6.0.0",
			"sha256-a5e2b223cb7c9c8efdc663ef484fdd95bb243bff242ef5b13e26883547fce9a0",
			core.Development,
			true,
		},
		"path": {
			"1.9.1",
			"sha256-75cca69d1490965be98c73ceaea117e8a04dd21217b37b292c9ddbec0d955bc5",
			core.Runtime,
			false,
		},
		"source_span": {
			"1.10.2",
			"sha256-56a02f1f4cd1a2d96303c0144c93bd6d909eea6bee6bf5a0e0b685edbd4c47ab",
			core.Runtime,
			false,
		},
		"string_scanner": {
			"1.4.1",
			"sha256-921cd31725b72fe181906c6a94d987c78e3b98c2e205b397ea399d4054872b43",
			core.Runtime,
			false,
		},
		"term_glyph": {
			"1.2.2",
			"sha256-7f554798625ea768a7518313e58f83891c7f5024f88e46e7182a4558850a4b8e",
			core.Runtime,
			false,
		},
		"yaml": {
			"3.1.3",
			"sha256-b9da305ac7c39faa3f030eccd175340f968459dae4af175130b3fc47e40d76ce",
			core.Runtime,
			true,
		},
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
		if dep.Integrity != want.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, want.integrity)
		}
		if dep.RegistryURL != "https://pub.dev" {
			t.Errorf("%s registry URL = %q, want %q", name, dep.RegistryURL, "https://pub.dev")
		}
		if dep.Scope != want.scope {
			t.Errorf("%s scope = %q, want %q", name, dep.Scope, want.scope)
		}
		if dep.Direct != want.direct {
			t.Errorf("%s direct = %t, want %t", name, dep.Direct, want.direct)
		}
	}
}

func TestPubspecLockLegacy(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pub/legacy/pubspec.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	res, err := (&pubspecLockParser{}).Parse("pubspec.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, dep := range res.Dependencies {
		depMap[dep.Name] = dep
	}

	expected := map[string]string{
		"analyzer": "0.24.6",
		"args":     "0.12.2+6",
		"barback":  "0.15.2+7",
		"which":    "0.1.3",
	}

	for name, version := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, version)
		}
		if dep.Scope != core.Runtime || dep.Integrity != "" || dep.RegistryURL != "" || dep.Direct {
			t.Errorf("unexpected metadata on legacy dependency: %+v", dep)
		}
	}
}
