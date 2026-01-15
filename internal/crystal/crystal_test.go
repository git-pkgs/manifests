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

	// Check runtime dependency with branch
	if dep, ok := depMap["frost"]; !ok {
		t.Error("expected frost dependency")
	} else {
		if dep.Version != "master" {
			t.Errorf("expected frost version master (from branch), got %s", dep.Version)
		}
		if dep.Scope != core.Runtime {
			t.Errorf("expected frost scope Runtime, got %v", dep.Scope)
		}
	}

	// Check runtime dependency with no version
	if dep, ok := depMap["shards"]; !ok {
		t.Error("expected shards dependency")
	} else if dep.Version != "" {
		t.Errorf("expected shards version empty, got %s", dep.Version)
	}

	// Check development dependency with version constraint
	if dep, ok := depMap["minitest"]; !ok {
		t.Error("expected minitest dependency")
	} else {
		if dep.Version != ">= 0.2.0" {
			t.Errorf("expected minitest version >= 0.2.0, got %s", dep.Version)
		}
		if dep.Scope != core.Development {
			t.Errorf("expected minitest scope Development, got %v", dep.Scope)
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

	// Check dependency with version
	if dep, ok := depMap["common_mark"]; !ok {
		t.Error("expected common_mark dependency")
	} else if dep.Version != "0.1.0" {
		t.Errorf("expected common_mark version 0.1.0, got %s", dep.Version)
	}

	// Check dependency with commit (no version)
	if dep, ok := depMap["frost"]; !ok {
		t.Error("expected frost dependency")
	} else if dep.Version != "4042fc55a0865df8cbcb9a389527e9557aa8f280" {
		t.Errorf("expected frost commit hash, got %s", dep.Version)
	}

	// Check minitest has version
	if dep, ok := depMap["minitest"]; !ok {
		t.Error("expected minitest dependency")
	} else if dep.Version != "0.3.1" {
		t.Errorf("expected minitest version 0.3.1, got %s", dep.Version)
	}
}
