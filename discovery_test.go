package manifests

import (
	"errors"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDiscoverManifestsRootAndKnownPaths(t *testing.T) {
	reader := mapFSReader(map[string]string{
		".github/workflows/ci.yaml": `jobs: {}`,
		".github/workflows/ci.yml":  `jobs: {}`,
		".github/workflows/readme":  "ignored",
		".gitmodules":               "",
		"README.md":                 "ignored",
		"nested/requirements.txt":   "requests==2.0.0",
		"package-lock.json":         `{"lockfileVersion": 3}`,
		"package.json":              `{"name":"root"}`,
	})

	got, warnings := DiscoverManifests(reader)
	if len(warnings) != 0 {
		t.Fatalf("DiscoverManifests warnings: %v", warnings)
	}
	want := []DiscoveredManifest{
		{Path: ".github/workflows/ci.yaml", Ecosystem: "github-actions", Kind: Manifest},
		{Path: ".github/workflows/ci.yml", Ecosystem: "github-actions", Kind: Manifest},
		{Path: ".gitmodules", Ecosystem: "git", Kind: Manifest},
		{Path: "package-lock.json", Ecosystem: "npm", Kind: Lockfile},
		{Path: "package.json", Ecosystem: "npm", Kind: Manifest},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("DiscoverManifests() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDiscoverManifestsWorkspaces(t *testing.T) {
	reader := mapFSReader(map[string]string{
		"Cargo.toml": `[workspace]
members = ["crates/*", "tools/cli"]
exclude = ["crates/private"]
`,
		"crates/core/Cargo.toml":    `[package]`,
		"crates/private/Cargo.toml": `[package]`,
		"tools/cli/Cargo.toml":      `[package]`,
		"go.work": `go 1.25.0

use (
    "./services/api"
    ./libs/shared
)
`,
		"services/api/go.mod":   `module example.com/api`,
		"libs/shared/go.mod":    `module example.com/shared`,
		"package.json":          `{"workspaces":{"packages":["apps/*"],"nohoist":["**"]}}`,
		"apps/web/package.json": `{"name":"web"}`,
		"pnpm-workspace.yaml": `packages:
  - "packages/**"
  - "!packages/**/fixtures"
`,
		"packages/direct/package.json":         `{"name":"direct"}`,
		"packages/group/api/package.json":      `{"name":"api"}`,
		"packages/group/fixtures/package.json": `{"name":"fixture"}`,
	})

	got, warnings := DiscoverManifests(reader)
	if len(warnings) != 0 {
		t.Fatalf("DiscoverManifests warnings: %v", warnings)
	}
	want := []DiscoveredManifest{
		{Path: "Cargo.toml", Ecosystem: "cargo", Kind: Manifest},
		{Path: "apps/web/package.json", Ecosystem: "npm", Kind: Manifest, ParentPath: "package.json"},
		{Path: "crates/core/Cargo.toml", Ecosystem: "cargo", Kind: Manifest, ParentPath: "Cargo.toml"},
		{Path: "libs/shared/go.mod", Ecosystem: "golang", Kind: Manifest, ParentPath: "go.work"},
		{Path: "package.json", Ecosystem: "npm", Kind: Manifest},
		{Path: "packages/direct/package.json", Ecosystem: "npm", Kind: Manifest, ParentPath: "pnpm-workspace.yaml"},
		{Path: "packages/group/api/package.json", Ecosystem: "npm", Kind: Manifest, ParentPath: "pnpm-workspace.yaml"},
		{Path: "services/api/go.mod", Ecosystem: "golang", Kind: Manifest, ParentPath: "go.work"},
		{Path: "tools/cli/Cargo.toml", Ecosystem: "cargo", Kind: Manifest, ParentPath: "Cargo.toml"},
	}
	if !slices.Equal(got, want) {
		t.Fatalf("DiscoverManifests() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestDiscoverManifestsNPMWorkspaceForms(t *testing.T) {
	tests := []struct {
		name       string
		workspaces string
	}{
		{name: "npm array", workspaces: `["packages/*"]`},
		{name: "yarn object", workspaces: `{"packages":["packages/*"],"nohoist":["**"]}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := mapFSReader(map[string]string{
				"package.json":              `{"workspaces":` + test.workspaces + `}`,
				"packages/api/package.json": `{"name":"api"}`,
			})

			got, warnings := DiscoverManifests(reader)
			if len(warnings) != 0 {
				t.Fatalf("DiscoverManifests warnings: %v", warnings)
			}
			want := []DiscoveredManifest{
				{Path: "package.json", Ecosystem: "npm", Kind: Manifest},
				{Path: "packages/api/package.json", Ecosystem: "npm", Kind: Manifest, ParentPath: "package.json"},
			}
			if !slices.Equal(got, want) {
				t.Fatalf("DiscoverManifests() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestDiscoverManifestsRejectsOutsideWorkspaceMembers(t *testing.T) {
	reader := mapFSReader(map[string]string{
		"Cargo.toml":              `[workspace]` + "\n" + `members = ["../outside", "/absolute"]`,
		"outside/Cargo.toml":      `[package]`,
		"absolute/Cargo.toml":     `[package]`,
		"nested/other/Cargo.toml": `[package]`,
	})

	got, warnings := DiscoverManifests(reader)
	if len(warnings) != 0 {
		t.Fatalf("DiscoverManifests warnings: %v", warnings)
	}
	want := []DiscoveredManifest{{Path: "Cargo.toml", Ecosystem: "cargo", Kind: Manifest}}
	if !slices.Equal(got, want) {
		t.Fatalf("DiscoverManifests() = %+v, want %+v", got, want)
	}
}

func TestDiscoverManifestsReturnsPartialResultsWithConfigurationWarnings(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		content   string
		wantError string
	}{
		{name: "cargo", path: "Cargo.toml", content: `[workspace`, wantError: "parsing Cargo workspace configuration"},
		{name: "npm", path: "package.json", content: `{"workspaces":true}`, wantError: "parsing npm workspace configuration"},
		{name: "pnpm", path: "pnpm-workspace.yaml", content: `packages: true`, wantError: "parsing pnpm workspace configuration"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, warnings := DiscoverManifests(mapFSReader(map[string]string{
				test.path:          test.content,
				"requirements.txt": "requests==2.0.0",
			}))
			if len(warnings) != 1 || !strings.Contains(warnings[0].Error(), test.wantError) {
				t.Fatalf("DiscoverManifests() warnings = %v, want one containing %q", warnings, test.wantError)
			}
			if !slices.Contains(got, DiscoveredManifest{Path: "requirements.txt", Ecosystem: "pypi", Kind: Manifest}) {
				t.Fatalf("DiscoverManifests() = %+v, want valid root manifest alongside warning", got)
			}
		})
	}
}

func TestDiscoverManifestsReaderWarnings(t *testing.T) {
	wantErr := errors.New("glob failed")
	_, warnings := DiscoverManifests(errorRepositoryReader{err: wantErr})
	if len(warnings) != 3 {
		t.Fatalf("DiscoverManifests() warnings = %v, want three root glob warnings", warnings)
	}
	for _, warning := range warnings {
		if !errors.Is(warning, wantErr) {
			t.Fatalf("DiscoverManifests() warning = %v, want wrapped %v", warning, wantErr)
		}
	}
}

func TestDiscoverManifestsWarnsOnMissingLiteralWorkspaceMember(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "go.work",
			files: map[string]string{
				"go.work":      "go 1.26\nuse (\n\t./svc-a\n\t./svc-b\n\t./missing\n)\n",
				"svc-a/go.mod": "module example.com/a",
				"svc-b/go.mod": "module example.com/b",
			},
			want: `go.work: workspace member "./missing" has no go.mod`,
		},
		{
			name: "cargo",
			files: map[string]string{
				"Cargo.toml":             "[workspace]\nmembers = [\"crates/core\", \"crates/gone\"]\n",
				"crates/core/Cargo.toml": "[package]",
			},
			want: `Cargo.toml: workspace member "crates/gone" has no Cargo.toml`,
		},
		{
			name: "npm literal",
			files: map[string]string{
				"package.json":          `{"workspaces":["apps/web","apps/gone"]}`,
				"apps/web/package.json": `{"name":"web"}`,
			},
			want: `package.json: workspace member "apps/gone" has no package.json`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, warnings := DiscoverManifests(mapFSReader(test.files))
			if len(warnings) != 1 || warnings[0].Error() != test.want {
				t.Fatalf("warnings = %v, want [%q]", warnings, test.want)
			}
			if len(got) < 2 {
				t.Errorf("valid members should still be returned, got %+v", got)
			}
		})
	}
}

func TestDiscoverManifestsNoWarningForEmptyWildcardMember(t *testing.T) {
	reader := mapFSReader(map[string]string{
		"go.work":      "go 1.26\nuse (\n\t./svc-a\n\t./extras/*\n)\n",
		"svc-a/go.mod": "module example.com/a",
	})
	got, warnings := DiscoverManifests(reader)
	if len(warnings) != 0 {
		t.Fatalf("wildcard with zero matches should not warn: %v", warnings)
	}
	want := []DiscoveredManifest{
		{Path: "svc-a/go.mod", Ecosystem: "golang", Kind: Manifest, ParentPath: "go.work"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDiscoverManifestsGoWorkspaceRootModule(t *testing.T) {
	reader := mapFSReader(map[string]string{
		"go.mod":  "module example.com/root\n",
		"go.work": "go 1.25.0\nuse .\n",
	})

	got, warnings := DiscoverManifests(reader)
	if len(warnings) != 0 {
		t.Fatalf("DiscoverManifests warnings: %v", warnings)
	}
	want := []DiscoveredManifest{{
		Path: "go.mod", Ecosystem: "golang", Kind: Manifest, ParentPath: "go.work",
	}}
	if !slices.Equal(got, want) {
		t.Fatalf("DiscoverManifests() = %+v, want %+v", got, want)
	}
}

func mapFSReader(files map[string]string) *FSReader {
	root := make(fstest.MapFS, len(files))
	for name, content := range files {
		root[name] = &fstest.MapFile{Data: []byte(content), Mode: 0o644}
	}
	return NewFSReader(root)
}

type errorRepositoryReader struct {
	err error
}

func (r errorRepositoryReader) ReadFile(string) ([]byte, error) {
	return nil, fs.ErrNotExist
}

func (r errorRepositoryReader) Glob(string) ([]string, error) {
	return nil, r.err
}
