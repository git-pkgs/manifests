package github_actions

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestActionsLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/github-actions/actions.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &actionsLockParser{}
	res, err := parser.Parse("actions.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 5 {
		t.Fatalf("expected 5 dependencies, got %d: %+v", len(res.Dependencies), res.Dependencies)
	}

	byKey := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		byKey[d.Name+"@"+d.Version] = d
	}

	checkout5, ok := byKey["actions/checkout@v5.0.1"]
	if !ok {
		t.Fatalf("expected actions/checkout@v5.0.1, got %+v", res.Dependencies)
	}
	if checkout5.Integrity != "sha1-93cb6efe18208431cddfb8368fd83d5badbf9bfd" {
		t.Errorf("checkout@v5.0.1 integrity = %q", checkout5.Integrity)
	}
	if !checkout5.Direct {
		t.Error("checkout@v5.0.1 should be direct (listed in workflows)")
	}
	if checkout5.Scope != core.Runtime {
		t.Errorf("checkout@v5.0.1 scope = %q, want runtime", checkout5.Scope)
	}

	if _, ok := byKey["actions/checkout@v6.0.2"]; !ok {
		t.Error("expected second actions/checkout entry at v6.0.2")
	}

	// actions/attest is only reached via cli/gh-extension-precompile's uses:
	// list, so it is transitive. Its pin key uses a SHA but ref: carries the
	// tag, which is what should surface as the version.
	attest, ok := byKey["actions/attest@v1.4.1"]
	if !ok {
		t.Fatalf("expected actions/attest@v1.4.1, got %+v", res.Dependencies)
	}
	if attest.Direct {
		t.Error("actions/attest should not be direct")
	}
}

func TestActionsLockIdentify(t *testing.T) {
	parser, eco, kind := core.IdentifyParser(".github/workflows/actions.lock")
	if parser == nil {
		t.Fatal("expected .github/workflows/actions.lock to be identified")
	}
	if eco != "github-actions" {
		t.Errorf("ecosystem = %q, want github-actions", eco)
	}
	if kind != core.Lockfile {
		t.Errorf("kind = %q, want lockfile", kind)
	}
}
