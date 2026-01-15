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

	// Check nixpkgs with ref in URL
	if dep, ok := depMap["nixpkgs"]; !ok {
		t.Error("expected nixpkgs dependency")
	} else if dep.Version != "nixos-unstable" {
		t.Errorf("expected nixpkgs version nixos-unstable, got %s", dep.Version)
	}

	// Check flake-utils
	if dep, ok := depMap["flake-utils"]; !ok {
		t.Error("expected flake-utils dependency")
	} else if dep.Version != "github:numtide/flake-utils" {
		t.Errorf("expected flake-utils version github:numtide/flake-utils, got %s", dep.Version)
	}

	// Check home-manager with ref
	if dep, ok := depMap["home-manager"]; !ok {
		t.Error("expected home-manager dependency")
	} else if dep.Version != "release-23.11" {
		t.Errorf("expected home-manager version release-23.11, got %s", dep.Version)
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check numtide/flake-utils
	if dep, ok := depMap["numtide/flake-utils"]; !ok {
		t.Error("expected numtide/flake-utils dependency")
	} else if dep.Version != "b1d9ab70662946ef0850d488da1c9019f3a9752a" {
		t.Errorf("expected flake-utils rev, got %s", dep.Version)
	}

	// Check NixOS/nixpkgs
	if dep, ok := depMap["NixOS/nixpkgs"]; !ok {
		t.Error("expected NixOS/nixpkgs dependency")
	} else if dep.Version != "44d0940ea560dee511026a53f0e2e2cde489b4d4" {
		t.Errorf("expected nixpkgs rev, got %s", dep.Version)
	}

	// Check nix-community/home-manager
	if dep, ok := depMap["nix-community/home-manager"]; !ok {
		t.Error("expected nix-community/home-manager dependency")
	} else if dep.Version != "f33900124c23c4eca5831b9b5eb32ea5894375ce" {
		t.Errorf("expected home-manager rev, got %s", dep.Version)
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

	// Check NixOS/nixpkgs
	if dep, ok := depMap["NixOS/nixpkgs"]; !ok {
		t.Error("expected NixOS/nixpkgs dependency")
	} else if dep.Version != "44d0940ea560dee511026a53f0e2e2cde489b4d4" {
		t.Errorf("nixpkgs version = %q", dep.Version)
	}

	// Check nix-community/home-manager
	if dep, ok := depMap["nix-community/home-manager"]; !ok {
		t.Error("expected nix-community/home-manager dependency")
	} else if dep.Version != "f33900124c23c4eca5831b9b5eb32ea5894375ce" {
		t.Errorf("home-manager version = %q", dep.Version)
	}
}
