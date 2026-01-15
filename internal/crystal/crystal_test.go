package crystal

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestShardYML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/crystal/shard.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &shardYMLParser{}
	deps, err := parser.Parse("shard.yml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 5 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"frost", "master", core.Runtime},
		{"shards", "", core.Runtime},
		{"common_mark", "", core.Runtime},
		{"minitest", ">= 0.2.0", core.Development},
		{"selenium-webdriver", "", core.Development},
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

func TestShardLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/crystal/shard.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &shardLockParser{}
	deps, err := parser.Parse("shard.lock", content)
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

	// All 7 packages with versions
	expected := map[string]string{
		"common_mark":       "0.1.0",
		"frost":             "4042fc55a0865df8cbcb9a389527e9557aa8f280",
		"minitest":          "0.3.1",
		"pg":                "0.5.0",
		"pool":              "0.2.1",
		"selenium-webdriver": "0.1.0",
		"shards":            "0.6.0",
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
