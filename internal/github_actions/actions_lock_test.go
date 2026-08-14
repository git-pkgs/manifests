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

func TestActionsLockV001(t *testing.T) {
	content, err := os.ReadFile("../../testdata/github-actions/actions-v0.0.1.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &actionsLockParser{}
	res, err := parser.Parse("actions.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got %d: %+v", len(res.Dependencies), res.Dependencies)
	}

	wantVersions := map[string]string{
		"actions/checkout": "v4",
		"example/branch":   "main",
		"example/fallback": "v2",
	}
	for _, dependency := range res.Dependencies {
		want, ok := wantVersions[dependency.Name]
		if !ok {
			t.Errorf("unexpected dependency: %+v", dependency)
			continue
		}
		if dependency.Version != want {
			t.Errorf("%s version = %q, want %q", dependency.Name, dependency.Version, want)
		}
	}
}

func TestActionsLockIdentify(t *testing.T) {
	tests := []struct {
		filename string
		want     bool
	}{
		{filename: ".github/workflows/actions.lock", want: true},
		{filename: "repo/.github/workflows/actions.lock", want: true},
		{filename: `.github\workflows\actions.lock`, want: true},
		{filename: `C:\repo\.github\workflows\actions.lock`, want: true},
		{filename: "actions.lock", want: false},
		{filename: "config/actions.lock", want: false},
		{filename: ".github/actions.lock", want: false},
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			parser, eco, kind := core.IdentifyParser(test.filename)
			if !test.want {
				if parser != nil {
					t.Fatalf("expected %s not to be identified, got %s %s", test.filename, eco, kind)
				}
				return
			}
			if parser == nil {
				t.Fatalf("expected %s to be identified", test.filename)
			}
			if eco != "github-actions" {
				t.Errorf("ecosystem = %q, want github-actions", eco)
			}
			if kind != core.Lockfile {
				t.Errorf("kind = %q, want lockfile", kind)
			}
		})
	}
}
