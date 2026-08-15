package hackage

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestCabal(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/example.cabal")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cabalParser{}
	res, err := parser.Parse("example.cabal", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 19 {
		t.Fatalf("expected 19 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// Sample of packages with versions (parser also picks up extensions)
	expected := map[string]string{
		"aeson":          "== 1.1.*",
		"base":           ">= 4.9 && < 4.11",
		"Cabal":          "== 2.0.*",
		"envy":           "== 1.3.*",
		"servant-server": "== 0.11.*",
		"text":           "== 1.2.*",
		"warp":           "== 3.2.*",
		"hspec":          "== 2.4.*",
		"bytestring":     "== 0.10.*",
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
	}
}

func TestStackLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/stack.yaml.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &stackLockParser{}
	res, err := parser.Parse("stack.yaml.lock", content)
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

	expected := map[string]struct {
		version   string
		integrity string
	}{
		"aeson": {
			"2.1.2.1",
			"sha256-5b8d62a60963a80d6d9d1abc29defbd4f5c57e5c9a4c3a1d4b1234567890abcd",
		},
		"text": {
			"2.0.2",
			"sha256-6c9d72b70783a80e6d6e2b1def29defbd4f5c57e5c9a4c3a1d4b1234567890ef",
		},
		"bytestring": {
			"0.11.5.3",
			"sha256-7d8e82c80893b90f7e3c2def29defbd4f5c57e5c9a4c3a1d4b12345678901234",
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
	}
}

func TestCabalConfig(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/cabal.config")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cabalConfigParser{}
	res, err := parser.Parse("cabal.config", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 108 {
		t.Fatalf("expected 108 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"aeson":      "1.1.2.0",
		"bytestring": "0.10.8.2",
		"base":       "4.10.1.0",
		"text":       "1.2.3.0",
		"warp":       "3.2.13",
		"Cabal":      "2.0.1.0",
	}

	for name, wantVer := range samples {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != wantVer {
			t.Errorf("%s version = %q, want %q", name, dep.Version, wantVer)
		}
	}
}

func TestCabalProjectFreeze(t *testing.T) {
	content, err := os.ReadFile("../../testdata/hackage/cabal.project.freeze")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cabalFreezeParser{}
	res, err := parser.Parse("cabal.project.freeze", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	// All 5 packages with versions
	expected := map[string]string{
		"Cabal":      "3.12.0.0",
		"aeson":      "2.2.3.0",
		"base":       "4.18.2.1",
		"bytestring": "0.11.5.3",
		"text":       "2.0.2",
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
	}
}
