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
	res, err := parser.Parse("flake.nix", content)
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

	// All 3 packages with exact versions
	expected := map[string]string{
		"nixpkgs":      "nixos-unstable",             // ref from URL path
		"flake-utils":  "github:numtide/flake-utils", // no ref, full URL as version
		"home-manager": "release-23.11",              // ref from URL path
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
	res, err := parser.Parse("flake.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	expected := map[string]struct {
		version   string
		integrity string
	}{
		"numtide/flake-utils": {
			"b1d9ab70662946ef0850d488da1c9019f3a9752a",
			"sha256-SZ5L6eA7HJ/nmkzGG7/ISclqe6oZdOZTNoesiInkXPQ=",
		},
		"nix-community/home-manager": {
			"f33900124c23c4eca5831b9b5eb32ea5894375ce",
			"sha256-s9Hi4RHhc6yut4EcYD50sZWRDKsugBJHOl3/Xq4xkDA=",
		},
		"NixOS/nixpkgs": {
			"44d0940ea560dee511026a53f0e2e2cde489b4d4",
			"sha256-YN/Ciidm+A0fmJPWlHBGvVkcarYWSC+s3NTPk/P+q3c=",
		},
		"nix-systems/default": {
			"da67096a3b9bf56a91d16901293e51ba5b49a27e",
			"sha256-Ber7lctkq3fmSDNldnNbWDmdjnAMQjn0XjhS0/vJwvo=",
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

func TestSourcesJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/nix/sources.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &sourcesJSONParser{}
	res, err := parser.Parse("sources.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d
	}

	expected := map[string]struct {
		version   string
		integrity string
	}{
		"NixOS/nixpkgs": {
			"44d0940ea560dee511026a53f0e2e2cde489b4d4",
			"sha256-1gp9qwyqwfv8x8k8rj11w7ajlb96a9njxsb5vg14dzadhn97slrc",
		},
		"nix-community/home-manager": {
			"f33900124c23c4eca5831b9b5eb32ea5894375ce",
			"sha256-0dfshsgj93ikfkcihf4c5z876h9wa319jqi84qkzc84r1amsnwq2",
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
