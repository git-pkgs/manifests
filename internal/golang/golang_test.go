package golang

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGoMod(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/go.mod")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &goModParser{}
	deps, err := parser.Parse("go.mod", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check indirect dependency
	if redigo, ok := depMap["github.com/gomodule/redigo"]; !ok {
		t.Error("expected github.com/gomodule/redigo dependency")
	} else {
		if redigo.Version != "v2.0.0+incompatible" {
			t.Errorf("redigo version = %q, want %q", redigo.Version, "v2.0.0+incompatible")
		}
		if redigo.Direct {
			t.Error("redigo should be indirect dependency")
		}
	}

	// Check direct dependency (no // indirect comment)
	if yaml, ok := depMap["gopkg.in/yaml.v1"]; !ok {
		t.Error("expected gopkg.in/yaml.v1 dependency")
	} else {
		if yaml.Version != "v1.0.0-20140924161607-9f9df34309c0" {
			t.Errorf("yaml version = %q, want %q", yaml.Version, "v1.0.0-20140924161607-9f9df34309c0")
		}
		if !yaml.Direct {
			t.Error("yaml should be direct dependency")
		}
	}

	// Check single-line require
	if net, ok := depMap["golang.org/x/net"]; !ok {
		t.Error("expected golang.org/x/net dependency")
	} else {
		if net.Version != "v1.2.3" {
			t.Errorf("net version = %q, want %q", net.Version, "v1.2.3")
		}
	}
}

func TestGoSum(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/go.sum")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &goSumParser{}
	deps, err := parser.Parse("go.sum", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check dependency with integrity hash
	if gocheck, ok := depMap["github.com/go-check/check"]; !ok {
		t.Error("expected github.com/go-check/check dependency")
	} else {
		if gocheck.Version != "v0.0.0-20180628173108-788fd7840127" {
			t.Errorf("go-check version = %q", gocheck.Version)
		}
		if gocheck.Integrity == "" {
			t.Error("expected go-check to have integrity hash")
		}
		if gocheck.Integrity != "h1:0gkP6mzaMqkmpcJYCFOLkIBwI7xFExG03bbkOkCvUPI=" {
			t.Errorf("go-check integrity = %q", gocheck.Integrity)
		}
	}

	// Verify we don't have /go.mod entries
	for name := range depMap {
		if name != "github.com/kr/pty" && depMap[name].Version == "v1.1.1/go.mod" {
			t.Errorf("should not have /go.mod entries, found %s %s", name, depMap[name].Version)
		}
	}
}

func TestGodepsJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Godeps.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &godepsJSONParser{}
	deps, err := parser.Parse("Godeps.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check BurntSushi/toml with version comment
	if dep, ok := depMap["github.com/BurntSushi/toml"]; !ok {
		t.Error("expected github.com/BurntSushi/toml dependency")
	} else if dep.Version != "v0.1.0-9-g3883ac1" {
		t.Errorf("toml version = %q, want %q", dep.Version, "v0.1.0-9-g3883ac1")
	}

	// Check gopkg.in/yaml.v1
	if _, ok := depMap["gopkg.in/yaml.v1"]; !ok {
		t.Error("expected gopkg.in/yaml.v1 dependency")
	}
}

func TestGlideYAML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/glide.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &glideYAMLParser{}
	deps, err := parser.Parse("glide.yaml", content)
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

	// Check Masterminds/vcs with version
	if dep, ok := depMap["github.com/Masterminds/vcs"]; !ok {
		t.Error("expected github.com/Masterminds/vcs dependency")
	} else if dep.Version != "^1.4.0" {
		t.Errorf("vcs version = %q, want %q", dep.Version, "^1.4.0")
	}

	// Check gopkg.in/yaml.v2 (no version)
	if dep, ok := depMap["gopkg.in/yaml.v2"]; !ok {
		t.Error("expected gopkg.in/yaml.v2 dependency")
	} else if dep.Version != "" {
		t.Errorf("yaml version = %q, want empty", dep.Version)
	}
}

func TestGlideLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/glide.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &glideLockParser{}
	deps, err := parser.Parse("glide.lock", content)
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

	// Check codegangsta/cli with commit hash
	if dep, ok := depMap["github.com/codegangsta/cli"]; !ok {
		t.Error("expected github.com/codegangsta/cli dependency")
	} else if dep.Version != "c31a7975863e7810c92e2e288a9ab074f9a88f29" {
		t.Errorf("cli version = %q", dep.Version)
	}
}

func TestGopkgTOML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Gopkg.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gopkgTOMLParser{}
	deps, err := parser.Parse("Gopkg.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check Masterminds/vcs with version
	if dep, ok := depMap["github.com/Masterminds/vcs"]; !ok {
		t.Error("expected github.com/Masterminds/vcs dependency")
	} else if dep.Version != "1.11.0" {
		t.Errorf("vcs version = %q, want %q", dep.Version, "1.11.0")
	}

	// Check branch constraint
	if dep, ok := depMap["github.com/pelletier/go-toml"]; !ok {
		t.Error("expected github.com/pelletier/go-toml dependency")
	} else if dep.Version != "master" {
		t.Errorf("go-toml version = %q, want %q", dep.Version, "master")
	}
}

func TestGopkgLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Gopkg.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gopkgLockParser{}
	deps, err := parser.Parse("Gopkg.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check boltdb/bolt with version
	if dep, ok := depMap["github.com/boltdb/bolt"]; !ok {
		t.Error("expected github.com/boltdb/bolt dependency")
	} else if dep.Version != "v1.3.1" {
		t.Errorf("bolt version = %q, want %q", dep.Version, "v1.3.1")
	}

	// Check pkg/errors with version
	if dep, ok := depMap["github.com/pkg/errors"]; !ok {
		t.Error("expected github.com/pkg/errors dependency")
	} else if dep.Version != "v0.8.0" {
		t.Errorf("errors version = %q, want %q", dep.Version, "v0.8.0")
	}
}

func TestVendorJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/vendor.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &vendorJSONParser{}
	deps, err := parser.Parse("vendor.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check pkg/errors
	if dep, ok := depMap["github.com/pkg/errors"]; !ok {
		t.Error("expected github.com/pkg/errors dependency")
	} else if dep.Version != "a2d6902c6d2a2f194eb3fb474981ab7867c81505" {
		t.Errorf("errors version = %q", dep.Version)
	}

	// Check Bowery/prompt
	if _, ok := depMap["github.com/Bowery/prompt"]; !ok {
		t.Error("expected github.com/Bowery/prompt dependency")
	}
}
