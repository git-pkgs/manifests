package manifests

import (
	"encoding/json"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/git-pkgs/manifests/internal/core"
)

// flixPackagesLock is the example lock file from Flix's Lockfile.scala. The
// manifest and archive digests of one package version share an algorithm.
// museum-clerk 0.4.0 records only its manifest.
const flixPackagesLock = `[lock]
version = 1

[packages."github:flix/museum"."1.2.3"]
toml    = "sha256:a4c123b1612dd272d1371c17149d439536b3216fdaeeb975729fae923d5a4fd1"
fpkg    = "sha256:2aabfe228f219e9cb0eb53f16947ccf25ec84d8dbc74254770f58904dba41ecc"

[packages."github:flix/museum-clerk"."0.4.0"]
toml    = "sha256:2ab1a5b8f6cbd0ba5b0e0b9f3b8d8b3d2b9c0e0a4a7f6f5e4d3c2b1a09f8e7d6"

[packages."github:flix/museum-clerk"."0.4.1"]
toml    = "sha256:cc3fc1626e53a13043b026c48bbf33feff9243a8f506b40928b5b7a767c76fb0"
fpkg    = "sha256:08f86bebb2737f6a6f0fb23c6f5da2cec255404e4fb440034d6608697a8d41be"
`

// artifactTestParser reads a Flix packages.lock into one dependency per
// package version, with a record for each recorded file. It stands in for a
// real Flix parser, so its entries carry no package identity. Records are
// appended archive last, with the archive repeated, so tests can check that
// Parse orders them and drops duplicates. The archive digest is also kept as
// the legacy Integrity value.
type artifactTestParser struct{}

func (artifactTestParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock struct {
		Packages map[string]map[string]struct {
			TOML string `toml:"toml"`
			FPKG string `toml:"fpkg"`
		} `toml:"packages"`
	}
	if _, err := toml.Decode(string(content), &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, id := range slices.Sorted(maps.Keys(lock.Packages)) {
		versions := lock.Packages[id]
		for _, version := range slices.Sorted(maps.Keys(versions)) {
			entry := versions[version]
			dep := core.Dependency{Name: id, Version: version, NoPURL: true}
			dep.Artifacts = append(dep.Artifacts, flixArtifact("toml", entry.TOML))
			if entry.FPKG != "" {
				archive := flixArtifact("fpkg", entry.FPKG)
				dep.Artifacts = append(dep.Artifacts, archive, archive)
				dep.Integrity = entry.FPKG
			}
			deps = append(deps, dep)
		}
	}
	return &core.Result{Dependencies: deps}, nil
}

func flixArtifact(artifact, value string) core.ArtifactIntegrity {
	algorithm, digest, _ := strings.Cut(value, ":")
	return core.ArtifactIntegrity{Artifact: artifact, Algorithm: algorithm, Digest: digest}
}

func init() {
	core.Register("flix", core.Supplement, artifactTestParser{}, core.ExactMatch("artifact-test.packages.lock"))
}

func TestArtifactIntegrityRecords(t *testing.T) {
	// https://github.com/git-pkgs/manifests/issues/112
	result, err := Parse("artifact-test.packages.lock", []byte(flixPackagesLock))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	type row struct {
		name, version, integrity string
		artifacts                []ArtifactIntegrity
	}
	want := []row{
		{"github:flix/museum", "1.2.3", "sha256:2aabfe228f219e9cb0eb53f16947ccf25ec84d8dbc74254770f58904dba41ecc", []ArtifactIntegrity{
			{Artifact: "fpkg", Algorithm: "sha256", Digest: "2aabfe228f219e9cb0eb53f16947ccf25ec84d8dbc74254770f58904dba41ecc"},
			{Artifact: "toml", Algorithm: "sha256", Digest: "a4c123b1612dd272d1371c17149d439536b3216fdaeeb975729fae923d5a4fd1"},
		}},
		{"github:flix/museum-clerk", "0.4.0", "", []ArtifactIntegrity{
			{Artifact: "toml", Algorithm: "sha256", Digest: "2ab1a5b8f6cbd0ba5b0e0b9f3b8d8b3d2b9c0e0a4a7f6f5e4d3c2b1a09f8e7d6"},
		}},
		{"github:flix/museum-clerk", "0.4.1", "sha256:08f86bebb2737f6a6f0fb23c6f5da2cec255404e4fb440034d6608697a8d41be", []ArtifactIntegrity{
			{Artifact: "fpkg", Algorithm: "sha256", Digest: "08f86bebb2737f6a6f0fb23c6f5da2cec255404e4fb440034d6608697a8d41be"},
			{Artifact: "toml", Algorithm: "sha256", Digest: "cc3fc1626e53a13043b026c48bbf33feff9243a8f506b40928b5b7a767c76fb0"},
		}},
	}

	// One row per package version, however many files it records.
	if len(result.Dependencies) != len(want) {
		t.Fatalf("got %d dependencies, want %d: %+v", len(result.Dependencies), len(want), result.Dependencies)
	}
	for i, dep := range result.Dependencies {
		w := want[i]
		if dep.Name != w.name || dep.Version != w.version {
			t.Errorf("dependency %d = %s@%s, want %s@%s", i, dep.Name, dep.Version, w.name, w.version)
		}
		if dep.Integrity != w.integrity {
			t.Errorf("%s@%s Integrity = %q, want %q", dep.Name, dep.Version, dep.Integrity, w.integrity)
		}
		if !reflect.DeepEqual(dep.Artifacts, w.artifacts) {
			t.Errorf("%s@%s Artifacts = %+v, want %+v", dep.Name, dep.Version, dep.Artifacts, w.artifacts)
		}
		for _, artifact := range dep.Artifacts {
			if artifact.Artifact == "toml" && strings.HasSuffix(dep.Integrity, artifact.Digest) {
				t.Errorf("%s@%s reports its manifest digest as Integrity", dep.Name, dep.Version)
			}
		}
	}
}

func TestArtifactIntegrityDeterministic(t *testing.T) {
	first, err := Parse("artifact-test.packages.lock", []byte(flixPackagesLock))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	for range 20 {
		again, err := Parse("artifact-test.packages.lock", []byte(flixPackagesLock))
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}
		if !reflect.DeepEqual(again.Dependencies, first.Dependencies) {
			t.Fatalf("Parse output changed between runs:\ngot  %+v\nwant %+v", again.Dependencies, first.Dependencies)
		}
	}
}

func TestArtifactIntegrityOptionalForExistingParsers(t *testing.T) {
	content, err := os.ReadFile("testdata/npm/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}
	result, err := Parse("package-lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	withIntegrity := 0
	for _, dep := range result.Dependencies {
		if dep.Artifacts != nil {
			t.Errorf("%s Artifacts = %+v, want nil", dep.Name, dep.Artifacts)
		}
		if dep.Integrity != "" {
			withIntegrity++
		}
	}
	if withIntegrity == 0 {
		t.Fatal("expected legacy Integrity values in package-lock.json")
	}

	encoded, err := json.Marshal(result.Dependencies[0])
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if strings.Contains(string(encoded), "Artifacts") {
		t.Errorf("JSON without artifact records includes Artifacts: %s", encoded)
	}
}

func TestArtifactIntegrityJSON(t *testing.T) {
	result, err := Parse("artifact-test.packages.lock", []byte(flixPackagesLock))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	encoded, err := json.Marshal(result.Dependencies[1])
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	want := `"Artifacts":[{"Artifact":"toml","Algorithm":"sha256","Digest":"2ab1a5b8f6cbd0ba5b0e0b9f3b8d8b3d2b9c0e0a4a7f6f5e4d3c2b1a09f8e7d6"}]`
	if !strings.Contains(string(encoded), want) {
		t.Errorf("JSON = %s, want it to contain %s", encoded, want)
	}
}
