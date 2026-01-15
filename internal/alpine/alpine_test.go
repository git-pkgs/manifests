package alpine

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestAPKBUILD(t *testing.T) {
	content, err := os.ReadFile("../../testdata/alpine/APKBUILD")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &apkbuildParser{}
	deps, err := parser.Parse("APKBUILD", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check runtime dependency
	if ca, ok := depMap["ca-certificates-bundle"]; !ok {
		t.Error("expected ca-certificates-bundle dependency")
	} else {
		if ca.Scope != core.Runtime {
			t.Errorf("ca-certificates-bundle scope = %q, want %q", ca.Scope, core.Runtime)
		}
	}

	// Check dev dependency with version
	if openssl, ok := depMap["openssl-dev"]; !ok {
		t.Error("expected openssl-dev dependency")
	} else {
		if openssl.Version != ">3" {
			t.Errorf("openssl-dev version = %q, want %q", openssl.Version, ">3")
		}
		if openssl.Scope != core.Development {
			t.Errorf("openssl-dev scope = %q, want %q", openssl.Scope, core.Development)
		}
	}

	// Check build dependency
	if perl, ok := depMap["perl"]; !ok {
		t.Error("expected perl dependency")
	} else {
		if perl.Scope != core.Build {
			t.Errorf("perl scope = %q, want %q", perl.Scope, core.Build)
		}
	}

	// Check test dependency
	if nghttp2, ok := depMap["nghttp2"]; !ok {
		t.Error("expected nghttp2 dependency")
	} else {
		if nghttp2.Scope != core.Test {
			t.Errorf("nghttp2 scope = %q, want %q", nghttp2.Scope, core.Test)
		}
	}
}

func TestAPKBUILDWithVersions(t *testing.T) {
	content, err := os.ReadFile("../../testdata/alpine/APKBUILD-with-versions")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &apkbuildParser{}
	deps, err := parser.Parse("APKBUILD", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check >= version constraint
	if libfoo, ok := depMap["libfoo"]; !ok {
		t.Error("expected libfoo dependency")
	} else {
		if libfoo.Version != ">=1.0" {
			t.Errorf("libfoo version = %q, want %q", libfoo.Version, ">=1.0")
		}
	}

	// Check < version constraint
	if libbar, ok := depMap["libbar"]; !ok {
		t.Error("expected libbar dependency")
	} else {
		if libbar.Version != "<2.0" {
			t.Errorf("libbar version = %q, want %q", libbar.Version, "<2.0")
		}
	}

	// Check no version constraint
	if libbaz, ok := depMap["libbaz"]; !ok {
		t.Error("expected libbaz dependency")
	} else {
		if libbaz.Version != "" {
			t.Errorf("libbaz version = %q, want empty", libbaz.Version)
		}
	}
}
