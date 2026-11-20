package manifests

import (
	"os"
	"reflect"
	"testing"
)

func parseFixture(t *testing.T, path, filename string) *ParseResult {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Parse(filename, content)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertReplacements(t *testing.T, got, want []Replacement) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("replacements = %d entries\n%+v\nwant %d\n%+v", len(got), got, len(want), want)
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("replacement %d =\n%+v\nwant\n%+v", i, got[i], want[i])
		}
	}
}

func TestPackageJSONReplacements(t *testing.T) {
	result := parseFixture(t, "testdata/npm/overrides/package.json", "package.json")

	if len(result.Dependencies) != 2 {
		t.Errorf("dependencies = %+v, want express and react only", result.Dependencies)
	}
	for _, dep := range result.Dependencies {
		if (dep.Name == "express" && dep.Version != "^4.18.2") || (dep.Name == "react" && dep.Version != "^18.2.0") {
			t.Errorf("dependency %s %s changed by an override", dep.Name, dep.Version)
		}
	}

	assertReplacements(t, result.Replacements, []Replacement{
		{Name: "semver", ReplacementVersion: "7.5.4", Location: "overrides/semver"},
		{Name: "react", ReplacementVersion: "$react", Location: "overrides/react"},
		{Name: "express", ReplacementVersion: "4.18.3", Location: "overrides/express/."},
		{Name: "cookie", ReplacementVersion: "0.6.0", Location: "overrides/express/cookie"},
		{Name: "debug", ReplacementName: "debug-fork", ReplacementVersion: "4.3.5", Location: "overrides/@babel%2Fcore@7.0.0/debug"},
		{Name: "minimist", ReplacementVersion: "1.2.8", Location: "resolutions/minimist"},
		{Name: "ws", Source: Source{Kind: SourceGitHub, Value: "websockets/ws", Ref: "8.17.1"}, Location: "resolutions/webpack%2F%2A%2A%2Fws"},
		{Name: "@types/node", Version: "^18", ReplacementVersion: "18.19.0", Location: "resolutions/@types%2Fnode@%5E18"},
		{Name: "lodash", ReplacementName: "lodash", ReplacementVersion: "4.17.21", Patch: "./.yarn/patches/lodash.patch", Location: "resolutions/lodash"},
		{Name: "qs", ReplacementVersion: "6.11.2", Location: "pnpm/overrides/qs"},
		{Name: "body-parser", Source: Source{Kind: SourcePath, Value: "file:../vendor/body-parser"}, Location: "pnpm/overrides/express@4%3Ebody-parser"},
		{Name: "bar", Version: "^1", ReplacementVersion: "-", Location: "pnpm/overrides/bar@%5E1"},
		{Name: "express", Version: "4.18.2", Patch: "patches/express@4.18.2.patch", Location: "pnpm/patchedDependencies/express@4.18.2"},
		{Name: "left-pad", Patch: "../shared/patches/left-pad.patch", Location: "pnpm/patchedDependencies/left-pad"},
	})
}

func TestPackageJSONReplacementsIgnoreMalformedBlocks(t *testing.T) {
	result, err := Parse("package.json", []byte(`{
  "dependencies": {"a": "1.0.0"},
  "overrides": ["not", "an", "object"],
  "resolutions": {"b": 2, "c": "1.0.0"},
  "pnpm": "nope"
}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 1 {
		t.Errorf("dependencies = %+v", result.Dependencies)
	}
	assertReplacements(t, result.Replacements, []Replacement{
		{Name: "c", ReplacementVersion: "1.0.0", Location: "resolutions/c"},
	})
}

func TestPackageJSONWithoutReplacements(t *testing.T) {
	result, err := Parse("package.json", []byte(`{"dependencies": {"a": "1.0.0"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if result.Replacements != nil {
		t.Errorf("replacements = %+v, want none", result.Replacements)
	}
}

func TestPnpmWorkspaceReplacements(t *testing.T) {
	result := parseFixture(t, "testdata/npm/overrides/pnpm-workspace.yaml", "pnpm-workspace.yaml")

	if result.Ecosystem != "npm" || result.Kind != Supplement {
		t.Errorf("ecosystem, kind = %s, %s", result.Ecosystem, result.Kind)
	}
	if len(result.Dependencies) != 0 {
		t.Errorf("dependencies = %+v, want none", result.Dependencies)
	}
	assertReplacements(t, result.Replacements, []Replacement{
		{Name: "foo", ReplacementVersion: "1.2.3", Location: "overrides/foo"},
		{Name: "child", ReplacementVersion: "^2.0.0", Location: "overrides/parent@1%3Echild"},
		{Name: "@scope/pkg", Version: "<1", ReplacementName: "@scope/fork", ReplacementVersion: "1.0.0", Location: "overrides/@scope%2Fpkg@%3C1"},
		{Name: "is-odd", Version: "3.0.1", Patch: "../../patches/is-odd@3.0.1.patch", Location: "patchedDependencies/is-odd@3.0.1"},
	})
}

func TestPnpmWorkspaceInvalidYAML(t *testing.T) {
	if _, err := Parse("pnpm-workspace.yaml", []byte("overrides: [")); err == nil {
		t.Error("expected an error for invalid YAML")
	}
}

func TestCargoReplacements(t *testing.T) {
	result := parseFixture(t, "testdata/cargo/patch/Cargo.toml", "Cargo.toml")

	for _, dep := range result.Dependencies {
		if dep.Source != (Source{}) {
			t.Errorf("dependency %s source = %+v, want none from a patch", dep.Name, dep.Source)
		}
	}
	assertReplacements(t, result.Replacements, []Replacement{
		{Name: "serde", Source: Source{Kind: SourceGit, Value: "https://github.com/serde-rs/serde", Branch: "fix"}, Location: "patch/crates-io/serde"},
		{Name: "rand", Source: Source{Kind: SourcePath, Value: "../rand"}, Location: "patch/crates-io/rand"},
		{Name: "uuid", ReplacementVersion: "1.8.0", Source: Source{Kind: SourcePath, Value: "../uuid"}, Location: "patch/crates-io/uuid"},
		{Name: "log", Source: Source{Kind: SourceGit, Value: "https://github.com/rust-lang/log", Ref: "abc123"}, Location: "patch/crates-io/my-log"},
		{Name: "baz", ReplacementVersion: "0.2.0", Source: Source{Kind: SourceRegistry, Value: "internal"}, Location: "patch/https:%2F%2Fgithub.com%2Fexample%2Fbaz/baz"},
		{Name: "foo", Version: "0.1.0", Source: Source{Kind: SourceGit, Value: "https://github.com/example/foo", Tag: "v0.1.1"}, Location: "replace/foo:0.1.0"},
		{Name: "bar", Version: "1.0.0", Source: Source{Kind: SourcePath, Value: "vendor/bar"}, Location: "replace/https:%2F%2Fgithub.com%2Frust-lang%2Fcrates.io-index%23bar@1.0.0"},
	})
}

func TestCargoWithoutReplacements(t *testing.T) {
	result, err := Parse("Cargo.toml", []byte("[package]\nname = \"app\"\n\n[dependencies]\nserde = \"1.0\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Replacements != nil {
		t.Errorf("replacements = %+v, want none", result.Replacements)
	}
}
