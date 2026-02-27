package ips

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestP5mParse(t *testing.T) {
	content, err := os.ReadFile("../../testdata/ips/sample.p5m")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &p5mParser{}
	deps, err := parser.Parse("sample.p5m", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 8 {
		for i, d := range deps {
			t.Logf("dep[%d]: %s %s %s", i, d.Name, d.Version, d.Scope)
		}
		t.Fatalf("expected 8 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Runtime dependencies
	for _, name := range []string{
		"library/libxml2",
		"system/library",
		"library/zlib",
		"library/openssl",
	} {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("missing dependency %s", name)
			continue
		}
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %q, want %q", name, dep.Scope, core.Runtime)
		}
		if !dep.Direct {
			t.Errorf("%s should be direct", name)
		}
	}

	// Optional dependencies
	for _, name := range []string{
		"developer/debug-tools",
		"library/python/setuptools-311",
	} {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("missing dependency %s", name)
			continue
		}
		if dep.Scope != core.Optional {
			t.Errorf("%s scope = %q, want %q", name, dep.Scope, core.Optional)
		}
	}

	// require-any alternatives
	for _, name := range []string{
		"web/server/apache-24",
		"web/server/nginx",
	} {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("missing require-any dependency %s", name)
			continue
		}
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %q, want %q", name, dep.Scope, core.Runtime)
		}
	}

	// Specific version checks
	if dep := depMap["library/libxml2"]; dep.Version != "2.9.14" {
		t.Errorf("libxml2 version = %q, want %q", dep.Version, "2.9.14")
	}
	if dep := depMap["system/library"]; dep.Version != "" {
		t.Errorf("system/library version = %q, want empty", dep.Version)
	}
	if dep := depMap["library/zlib"]; dep.Version != "1.2.13" {
		t.Errorf("zlib version = %q, want %q", dep.Version, "1.2.13")
	}
	if dep := depMap["library/openssl"]; dep.Version != "3.1.4" {
		t.Errorf("openssl version = %q, want %q", dep.Version, "3.1.4")
	}
	if dep := depMap["web/server/nginx"]; dep.Version != "1.24.0" {
		t.Errorf("nginx version = %q, want %q", dep.Version, "1.24.0")
	}
}

func TestP5mSkipsMacros(t *testing.T) {
	content := []byte(`depend type=require fmri=pkg:/library/$(MACH64)/libfoo@1.0
depend type=require fmri=__TBD pkg.debug.depend.file=libbar.so
depend type=require fmri=pkg:/library/libxml2@2.9
`)

	parser := &p5mParser{}
	deps, err := parser.Parse("test.p5m", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency (skipping macro and TBD), got %d", len(deps))
	}
	if deps[0].Name != "library/libxml2" {
		t.Errorf("expected library/libxml2, got %s", deps[0].Name)
	}
}

func TestP5mMultilineContinuation(t *testing.T) {
	content := []byte(`depend type=require-any \
    fmri=pkg:/editor/vim@9.0 \
    fmri=pkg:/editor/neovim@0.9
`)

	parser := &p5mParser{}
	deps, err := parser.Parse("test.p5m", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies from require-any, got %d", len(deps))
	}

	names := map[string]string{
		"editor/vim":    "9.0",
		"editor/neovim": "0.9",
	}
	for _, dep := range deps {
		wantVer, ok := names[dep.Name]
		if !ok {
			t.Errorf("unexpected dependency %s", dep.Name)
			continue
		}
		if dep.Version != wantVer {
			t.Errorf("%s version = %q, want %q", dep.Name, dep.Version, wantVer)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %q, want %q", dep.Name, dep.Scope, core.Runtime)
		}
	}
}

func TestParseFMRI(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"pkg:/library/zlib@1.2.13,5.11-2024.0.0.0", "library/zlib", "1.2.13"},
		{"pkg:/library/libxml2@2.9.14", "library/libxml2", "2.9.14"},
		{"pkg:/system/library", "system/library", ""},
		{"library/openssl@3.1", "library/openssl", "3.1"},
		{"pkg://openindiana.org/library/glib2@2.76,5.11-2024.0.0.0", "library/glib2", "2.76"},
	}

	for _, tt := range tests {
		name, version := parseFMRI(tt.input)
		if name != tt.wantName {
			t.Errorf("parseFMRI(%q) name = %q, want %q", tt.input, name, tt.wantName)
		}
		if version != tt.wantVersion {
			t.Errorf("parseFMRI(%q) version = %q, want %q", tt.input, version, tt.wantVersion)
		}
	}
}
