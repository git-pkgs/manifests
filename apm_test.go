package manifests

import (
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestAPMIdentifyAndDiscover(t *testing.T) {
	for filename, kind := range map[string]Kind{"apm.yml": Manifest, "apm.lock.yaml": Lockfile, "apm.lock": Lockfile} {
		ecosystem, gotKind, ok := Identify(filename)
		if !ok || ecosystem != "apm" || gotKind != kind {
			t.Errorf("Identify(%q) = %q, %q, %v; want apm %s", filename, ecosystem, gotKind, ok, kind)
		}
	}

	got, warnings := DiscoverManifests(mapFSReader(map[string]string{
		"apm.yml":       "name: x\nversion: 1.0.0\n",
		"apm.lock.yaml": "lockfile_version: \"1\"\ndependencies: []\n",
	}))
	if len(warnings) != 0 {
		t.Fatalf("DiscoverManifests warnings: %v", warnings)
	}
	want := []DiscoveredManifest{
		{Path: "apm.lock.yaml", Ecosystem: "apm", Kind: Lockfile},
		{Path: "apm.yml", Ecosystem: "apm", Kind: Manifest},
	}
	if !slices.Equal(got, want) {
		t.Errorf("DiscoverManifests() = %+v, want %+v", got, want)
	}
}

type apmRow struct {
	version, purl, ecosystem string
	scope                    Scope
	direct                   bool
	source                   Source
}

func checkAPMRows(t *testing.T, deps []Dependency, want map[string]apmRow) {
	t.Helper()
	if len(deps) != len(want) {
		t.Fatalf("got %d dependencies, want %d: %+v", len(deps), len(want), deps)
	}
	for _, dep := range deps {
		w, ok := want[dep.Name]
		if !ok {
			t.Errorf("unexpected dependency %q", dep.Name)
			continue
		}
		got := apmRow{dep.Version, dep.PURL, dep.Ecosystem, dep.Scope, dep.Direct, dep.Source}
		if got != w {
			t.Errorf("%s =\n%+v\nwant\n%+v", dep.Name, got, w)
		}
		if dep.NoPURL != (w.purl == "") {
			t.Errorf("%s NoPURL = %v, want %v", dep.Name, dep.NoPURL, w.purl == "")
		}
	}
}

func TestAPMManifest(t *testing.T) {
	content, err := os.ReadFile("testdata/apm/apm.yml")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Parse("apm.yml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Ecosystem != "apm" || result.Kind != Manifest || result.Name != "agent-workspace" || result.Version != "1.4.0" {
		t.Errorf("result = %s %s %q %q", result.Ecosystem, result.Kind, result.Name, result.Version)
	}
	if !slices.Equal(result.Licenses, []string{"MIT"}) {
		t.Errorf("Licenses = %v, want [MIT]", result.Licenses)
	}
	if len(result.Scripts) != 2 || result.Scripts["start"][0] != "copilot -p 'README.prompt.md'" {
		t.Errorf("Scripts = %v", result.Scripts)
	}

	git := func(value, ref string) Source { return Source{Kind: SourceGit, Value: value, Ref: ref} }
	checkAPMRows(t, result.Dependencies, map[string]apmRow{
		"github.com/microsoft/apm-sample-package": {"v1.0.0", "pkg:git/github.com/microsoft/apm-sample-package", "git", Runtime, true,
			git("microsoft/apm-sample-package", "v1.0.0")},
		"github.com/composiohq/awesome-claude-skills/brand-guidelines": {"", "pkg:git/github.com/composiohq/awesome-claude-skills#brand-guidelines", "git", Runtime, true,
			git("ComposioHQ/awesome-claude-skills/brand-guidelines", "")},
		"github.com/contoso/prompts/review.prompt.md": {"", "pkg:git/github.com/contoso/prompts#review.prompt.md", "git", Runtime, true,
			git("contoso/prompts/review.prompt.md", "")},
		"gitlab.com/acme/coding-standards": {"main", "pkg:git/gitlab.com/acme/coding-standards", "git", Runtime, true,
			git("gitlab.com/acme/coding-standards", "main")},
		"github.com/acme-corp/security-baseline": {"", "pkg:git/github.com/acme-corp/security-baseline", "git", Runtime, true,
			git("https://github.com/Acme-Corp/Security-Baseline.git", "")},
		"gitlab.com/acme/repo/instructions/security": {"v2.0", "pkg:git/gitlab.com/acme/repo#instructions/security", "git", Runtime, true,
			git("https://gitlab.com/acme/repo.git", "v2.0")},
		"./packages/shared-skills":           {"", "", "", Runtime, true, Source{Kind: SourcePath, Value: "./packages/shared-skills"}},
		"./ai/agents/reviewer.agent.md":      {"", "", "", Runtime, true, Source{Kind: SourcePath, Value: "./ai/agents/reviewer.agent.md"}},
		"sec-check":                          {"~2.1.0", "", "", Runtime, true, Source{Kind: SourceRegistry, Value: "acme-plugins"}},
		"acme/toolkit":                       {"^2.0.0", "", "", Runtime, true, Source{Kind: SourceRegistry, Value: "https://artifactory.example.com/artifactory/api/skills/jf-skills-local"}},
		"io.github.github/github-mcp-server": {"", "", "", Runtime, true, Source{Kind: SourceRegistry}},
		"my-private-server":                  {"", "", "", Runtime, true, Source{}},
		"slack":                              {"", "", "", Runtime, true, Source{Kind: SourceURL, Value: "https://mcp.slack.com/mcp"}},
		"gopls":                              {"", "", "", Runtime, true, Source{}},
		"pyright":                            {"", "", "", Runtime, true, Source{}},
		"github.com/owner/test-helpers":      {"", "pkg:git/github.com/owner/test-helpers", "git", Development, true, git("owner/test-helpers", "")},
		"../sibling-repo/lint-rules":         {"", "", "", Development, true, Source{Kind: SourcePath, Value: "../sibling-repo/lint-rules"}},
	})
}

func TestAPMManifestDefaultRegistryShorthand(t *testing.T) {
	result, err := Parse("apm.yml", []byte(`name: x
version: 1.0.0
registries:
  corp:
    url: https://registry.example.com
  mirror: https://mirror.example.com
  default: corp
dependencies:
  apm:
    - acme/toolkit#^2.0.0
    - acme/pinned
    - id: acme/mirrored
      registry: mirror
      version: 1.0.0
    - id: acme/user-level
      registry: user-registry
      version: 1.0.0
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checkAPMRows(t, result.Dependencies, map[string]apmRow{
		"acme/toolkit":  {"^2.0.0", "", "", Runtime, true, Source{Kind: SourceRegistry, Value: "https://registry.example.com"}},
		"acme/mirrored": {"1.0.0", "", "", Runtime, true, Source{Kind: SourceRegistry, Value: "https://mirror.example.com"}},
		// Registries declared only in user config cannot be resolved offline.
		"acme/user-level":        {"1.0.0", "", "", Runtime, true, Source{Kind: SourceRegistry, Value: "user-registry"}},
		"github.com/acme/pinned": {"", "pkg:git/github.com/acme/pinned", "git", Runtime, true, Source{Kind: SourceGit, Value: "acme/pinned"}},
	})
}

func TestAPMLockfile(t *testing.T) {
	content, err := os.ReadFile("testdata/apm/apm.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Parse("apm.lock.yaml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	const skillsCommit = "7f3c9a4d2e1b8c7f0a9e6d5c4b3a2918f7e6d5c4"
	skills := Source{Kind: SourceGit, Value: "ComposioHQ/awesome-claude-skills", Ref: "main"}
	checkAPMRows(t, result.Dependencies, map[string]apmRow{
		"github.com/microsoft/apm-sample-package": {"a1b2c3d4e5f6789012345678901234567890abcd",
			"pkg:git/github.com/microsoft/apm-sample-package@a1b2c3d4e5f6789012345678901234567890abcd", "git", Runtime, true,
			Source{Kind: SourceGit, Value: "microsoft/apm-sample-package", Ref: "v1.0.0"}},
		// Two subpackages of one repository at one commit stay distinct.
		"github.com/composiohq/awesome-claude-skills/brand-guidelines": {skillsCommit,
			"pkg:git/github.com/composiohq/awesome-claude-skills@" + skillsCommit + "#brand-guidelines", "git", Runtime, true, skills},
		"github.com/composiohq/awesome-claude-skills/canvas-design": {skillsCommit,
			"pkg:git/github.com/composiohq/awesome-claude-skills@" + skillsCommit + "#canvas-design", "git", Runtime, true, skills},
		"gitlab.com/acme/coding-standards": {"f6e5d4c3b2a1098765432109876543210fedcba9",
			"pkg:git/gitlab.com/acme/coding-standards@f6e5d4c3b2a1098765432109876543210fedcba9", "git", Runtime, false,
			Source{Kind: SourceGit, Value: "acme/coding-standards", Ref: "main"}},
		"acme/toolkit": {"2.3.1", "", "", Runtime, true,
			Source{Kind: SourceRegistry, Value: "https://registry.example.com/v1/packages/acme/toolkit/versions/2.3.1/download"}},
		"./packages/shared-skills": {"0.1.0", "", "", Runtime, true, Source{Kind: SourcePath, Value: "./packages/shared-skills"}},
		"github.com/owner/test-helpers": {"0123456789abcdef0123456789abcdef01234567",
			"pkg:git/github.com/owner/test-helpers@0123456789abcdef0123456789abcdef01234567", "git", Development, true,
			Source{Kind: SourceGit, Value: "owner/test-helpers"}},
	})

	byName := map[string]Dependency{}
	for _, dep := range result.Dependencies {
		byName[dep.Name] = dep
	}
	sha := func(artifact, digest string) ArtifactIntegrity {
		return ArtifactIntegrity{Artifact: artifact, Algorithm: "sha256", Digest: digest}
	}
	artifacts := map[string][]ArtifactIntegrity{
		"github.com/microsoft/apm-sample-package": {
			sha("deployed:.github/instructions/sample.instructions.md", "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae"),
			sha("tree", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"),
		},
		"acme/toolkit":             {sha("archive", "fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9")},
		"./packages/shared-skills": {sha("tree", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")},
	}
	for name, dep := range byName {
		if !reflect.DeepEqual(dep.Artifacts, artifacts[name]) {
			t.Errorf("%s Artifacts = %+v, want %+v", name, dep.Artifacts, artifacts[name])
		}
	}

	// Only the registry archive digest is reported as Integrity.
	for name, dep := range byName {
		want := ""
		if name == "acme/toolkit" {
			want = "sha256-fcde2b2edba56bf408601fb721fe9b5c338d10ee429ea04fae5511b68fbf8fb9"
		}
		if dep.Integrity != want {
			t.Errorf("%s Integrity = %q, want %q", name, dep.Integrity, want)
		}
	}
}

func TestAPMLockfileCommitOnlyUpdate(t *testing.T) {
	content, err := os.ReadFile("testdata/apm/apm.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	const oldCommit, newCommit = "a1b2c3d4e5f6789012345678901234567890abcd", "b2c3d4e5f6789012345678901234567890abcdef"
	updated := strings.Replace(string(content), oldCommit, newCommit, 1)

	find := func(content string) Dependency {
		result, err := Parse("apm.lock.yaml", []byte(content))
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		for _, dep := range result.Dependencies {
			if dep.Name == "github.com/microsoft/apm-sample-package" {
				return dep
			}
		}
		t.Fatal("apm-sample-package not found")
		return Dependency{}
	}
	before, after := find(string(content)), find(updated)
	if before.Version == after.Version || before.PURL == after.PURL {
		t.Errorf("commit-only update not visible: before %q %q, after %q %q", before.Version, before.PURL, after.Version, after.PURL)
	}
	if after.Version != newCommit {
		t.Errorf("Version = %q, want %q", after.Version, newCommit)
	}
}

func TestAPMLegacyLockfile(t *testing.T) {
	content, err := os.ReadFile("testdata/apm/apm.lock.yaml")
	if err != nil {
		t.Fatal(err)
	}
	current, err := Parse("apm.lock.yaml", content)
	if err != nil {
		t.Fatalf("Parse apm.lock.yaml: %v", err)
	}
	legacy, err := Parse("apm.lock", content)
	if err != nil {
		t.Fatalf("Parse apm.lock: %v", err)
	}
	if legacy.Kind != Lockfile || !reflect.DeepEqual(legacy.Dependencies, current.Dependencies) {
		t.Errorf("legacy apm.lock differs from apm.lock.yaml")
	}
}

func TestAPMLockfileVersions(t *testing.T) {
	entry := "dependencies:\n  - repo_url: owner/repo\n    resolved_commit: 0123456789abcdef0123456789abcdef01234567\n"
	tests := []struct {
		name, content string
		wantErr       bool
	}{
		{"pre-versioned", entry, false},
		{"version 1", "lockfile_version: \"1\"\n" + entry, false},
		{"version 2", "lockfile_version: \"2\"\n" + entry, false},
		{"candidate replacement version", "lockfile_version: \"3\"\nreplacements: []\n" + entry, true},
		{"replacements without version 3", "lockfile_version: \"2\"\nreplacements: []\n" + entry, true},
		{"future version", "lockfile_version: \"9\"\n" + entry, true},
		{"unquoted version", "lockfile_version: 1\n" + entry, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse("apm.lock.yaml", []byte(tt.content))
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAPMMalformed(t *testing.T) {
	tests := []struct{ filename, content string }{
		{"apm.yml", "name: [unterminated\n"},
		{"apm.yml", "name: x\ndependencies:\n  - owner/repo\n"},
		{"apm.lock.yaml", "lockfile_version: \"1\"\ndependencies: {}\n"},
		{"apm.lock.yaml", "- not a mapping\n"},
		{"apm.lock.yaml", "lockfile_version: \"1\"\ndependencies:\n  - resolved_commit: abc\n"},
		{"apm.lock", "<<<<<<< HEAD\nlockfile_version: \"1\"\n"},
	}
	for _, tt := range tests {
		if _, err := Parse(tt.filename, []byte(tt.content)); err == nil {
			t.Errorf("Parse(%s, %q) succeeded, want error", tt.filename, tt.content)
		}
	}
}

func TestAPMLockfileExcludesSelf(t *testing.T) {
	result, err := Parse("apm.lock.yaml", []byte(`lockfile_version: "1"
dependencies:
  - repo_url: <self>
    source: local
    local_path: .
    depth: 0
  - repo_url: owner/root-depth
    depth: 0
  - repo_url: owner/repo
    resolved_commit: 0123456789abcdef0123456789abcdef01234567
`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "github.com/owner/repo" {
		t.Errorf("Dependencies = %+v, want only github.com/owner/repo", result.Dependencies)
	}
}

func TestAPMManifestAndLockfileIdentitiesMatch(t *testing.T) {
	manifest, err := Parse("apm.yml", []byte(`name: x
version: 1.0.0
dependencies:
  apm:
    - contoso.ghe.com/Owner/Repo/skills/x
    - git@ssh.dev.azure.com:v3/org/project/repo
    - https://org.visualstudio.com/project/_git/repo
`))
	if err != nil {
		t.Fatalf("Parse apm.yml: %v", err)
	}
	lock, err := Parse("apm.lock.yaml", []byte(`lockfile_version: "1"
dependencies:
  - repo_url: owner/repo
    host: contoso.ghe.com
    virtual_path: skills/x
    is_virtual: true
    resolved_commit: 0123456789abcdef0123456789abcdef01234567
  - repo_url: org/project/repo
    host: ssh.dev.azure.com
    resolved_commit: 0123456789abcdef0123456789abcdef01234567
  - repo_url: project/repo
    host: org.visualstudio.com
    resolved_commit: 0123456789abcdef0123456789abcdef01234567
`))
	if err != nil {
		t.Fatalf("Parse apm.lock.yaml: %v", err)
	}

	want := []string{
		"pkg:git/contoso.ghe.com/owner/repo#skills/x",
		"pkg:git/ssh.dev.azure.com/org/project/repo",
		"pkg:git/org.visualstudio.com/project/repo",
	}
	for i, w := range want {
		if got := manifest.Dependencies[i].PURL; got != w {
			t.Errorf("manifest PURL = %q, want %q", got, w)
		}
		// Lockfile PURLs add the commit as the version.
		if got := strings.Replace(lock.Dependencies[i].PURL, "@0123456789abcdef0123456789abcdef01234567", "", 1); got != w {
			t.Errorf("lockfile PURL = %q, want %q", lock.Dependencies[i].PURL, w)
		}
		if manifest.Dependencies[i].Name != lock.Dependencies[i].Name {
			t.Errorf("Name %q in manifest, %q in lockfile", manifest.Dependencies[i].Name, lock.Dependencies[i].Name)
		}
	}
}
