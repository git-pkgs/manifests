package guix

import (
	"os"
	"testing"
)

func TestManifest(t *testing.T) {
	content, err := os.ReadFile("../../testdata/guix/manifest.scm")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &manifestParser{}
	deps, err := parser.Parse("manifest.scm", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 15 {
		t.Fatalf("expected 15 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]bool)
	for _, d := range deps {
		depMap[d.Name] = true
		if d.Scope != "runtime" {
			t.Errorf("%s: scope = %q, want %q", d.Name, d.Scope, "runtime")
		}
		if !d.Direct {
			t.Errorf("%s: expected direct", d.Name)
		}
	}

	expected := []string{
		"bash", "coreutils", "gcc-toolchain", "git-minimal", "grep",
		"gzip", "make", "nss-certs", "pkg-config", "python",
		"sed", "tar", "util-linux", "wget", "xz",
	}
	for _, name := range expected {
		if !depMap[name] {
			t.Errorf("expected %s dependency", name)
		}
	}

	// Verify comment was filtered
	if depMap["commented-out-package"] {
		t.Error("commented-out-package should not be included")
	}
}

func TestManifestWithVersions(t *testing.T) {
	content := []byte(`(specifications->manifest
 '("python@3.10"
   "node@18.0.0"
   "go"))
`)

	parser := &manifestParser{}
	deps, err := parser.Parse("manifest.scm", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	cases := map[string]string{
		"python": "3.10",
		"node":   "18.0.0",
		"go":     "",
	}

	for _, d := range deps {
		want, ok := cases[d.Name]
		if !ok {
			t.Errorf("unexpected dependency %s", d.Name)
			continue
		}
		if d.Version != want {
			t.Errorf("%s: version = %q, want %q", d.Name, d.Version, want)
		}
	}
}

func TestManifestNoMatch(t *testing.T) {
	// File with no specifications->manifest call returns nil
	content := []byte(`(use-modules (gnu packages))
(packages->manifest (list some-binding))
`)

	parser := &manifestParser{}
	deps, err := parser.Parse("manifest.scm", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if deps != nil {
		t.Fatalf("expected nil dependencies for non-specifications manifest, got %d", len(deps))
	}
}

func TestManifestSuffixMatch(t *testing.T) {
	content := []byte(`(specifications->manifest '("git" "make"))`)

	parser := &manifestParser{}
	deps, err := parser.Parse("base-manifest.scm", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}
}
