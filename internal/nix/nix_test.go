package nix

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestFlakeNix(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nix/flake.nix")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &flakeNixParser{}
	deps, err := parser.Parse("flake.nix", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 3 packages with exact versions
	expected := map[string]string{
		"nixpkgs":      "nixos-unstable",           // ref from URL path
		"flake-utils":  "github:numtide/flake-utils", // no ref, full URL as version
		"home-manager": "release-23.11",            // ref from URL path
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

func TestFlakeLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nix/flake.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &flakeLockParser{}
	deps, err := parser.Parse("flake.lock", content)
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

	// All 4 packages with exact revisions
	// Note: parser doesn't currently extract narHash as integrity
	expected := map[string]string{
		"numtide/flake-utils":        "b1d9ab70662946ef0850d488da1c9019f3a9752a",
		"nix-community/home-manager": "f33900124c23c4eca5831b9b5eb32ea5894375ce",
		"NixOS/nixpkgs":              "44d0940ea560dee511026a53f0e2e2cde489b4d4",
		"nix-systems/default":        "da67096a3b9bf56a91d16901293e51ba5b49a27e",
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

func TestSourcesJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nix/sources.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &sourcesJSONParser{}
	deps, err := parser.Parse("sources.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 2 packages with exact revisions
	// Note: parser doesn't currently extract sha256 as integrity
	expected := map[string]string{
		"NixOS/nixpkgs":              "44d0940ea560dee511026a53f0e2e2cde489b4d4",
		"nix-community/home-manager": "f33900124c23c4eca5831b9b5eb32ea5894375ce",
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
