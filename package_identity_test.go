package manifests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

// identityTestParser returns one entry for each way a parser can describe a
// package identity, so tests can check how Parse resolves them.
type identityTestParser struct{}

func (identityTestParser) Parse(string, []byte) (*core.Result, error) {
	jar := Source{Kind: SourceURL, Value: "https://example.com/lib.jar"}
	return &core.Result{
		Dependencies: []core.Dependency{
			{Name: "org.example:fallback", Version: "1.0.0"},
			{Name: "lodash", Version: "4.17.21", Ecosystem: "npm"},
			{Name: "tool", Version: "2.0.0", PURL: "pkg:generic/tool@2.0.0", Ecosystem: "generic"},
			{Name: "lib.jar", Version: "3.0.0", NoPURL: true, Source: jar},
			{Name: "ignored", Version: "4.0.0", NoPURL: true, PURL: "pkg:generic/ignored@4.0.0"},
		},
		Declarations: []core.Declaration{
			{Name: "org.example:fallback", Version: "1.0.0", Location: "fallback"},
			{Name: "lodash", Version: "4.17.21", Ecosystem: "npm", Location: "lodash"},
			{Name: "tool", Version: "2.0.0", PURL: "pkg:generic/tool", Ecosystem: "generic", Location: "tool"},
			{Name: "lib.jar", Version: "3.0.0", NoPURL: true, Source: jar, Location: "lib.jar"},
			{Name: "ignored", Version: "4.0.0", NoPURL: true, PURL: "pkg:generic/ignored", Location: "ignored"},
		},
	}, nil
}

func init() {
	core.Register("maven", core.Manifest, identityTestParser{}, core.ExactMatch("identity-test.manifest"))
	core.Register("maven", core.Lockfile, identityTestParser{}, core.ExactMatch("identity-test.lock"))
}

// identityTestEcosystems lists the ecosystem Parse resolves for each entry
// returned by identityTestParser. NoPURL entries have no ecosystem.
var identityTestEcosystems = []string{"maven", "npm", "generic", "", ""}

func TestDependencyPackageIdentity(t *testing.T) {
	// https://github.com/git-pkgs/manifests/issues/111
	tests := []struct {
		filename string
		want     []string
	}{
		{"identity-test.manifest", []string{
			"pkg:maven/org.example/fallback",
			"pkg:npm/lodash",
			"pkg:generic/tool@2.0.0",
			"",
			"",
		}},
		{"identity-test.lock", []string{
			"pkg:maven/org.example/fallback@1.0.0",
			"pkg:npm/lodash@4.17.21",
			"pkg:generic/tool@2.0.0",
			"",
			"",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result, err := Parse(tt.filename, nil)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if result.Ecosystem != "maven" {
				t.Errorf("Ecosystem = %q, want maven", result.Ecosystem)
			}
			if len(result.Dependencies) != len(tt.want) {
				t.Fatalf("got %d dependencies, want %d", len(result.Dependencies), len(tt.want))
			}
			for i, dep := range result.Dependencies {
				if dep.PURL != tt.want[i] {
					t.Errorf("%s PURL = %q, want %q", dep.Name, dep.PURL, tt.want[i])
				}
				if dep.Ecosystem != identityTestEcosystems[i] {
					t.Errorf("%s Ecosystem = %q, want %q", dep.Name, dep.Ecosystem, identityTestEcosystems[i])
				}
			}
			if got := result.Dependencies[3].Source.Value; got != "https://example.com/lib.jar" {
				t.Errorf("lib.jar Source.Value = %q, want the download URL", got)
			}
		})
	}
}

func TestDeclarationPackageIdentity(t *testing.T) {
	// https://github.com/git-pkgs/manifests/issues/111
	want := []string{
		"pkg:maven/org.example/fallback",
		"pkg:npm/lodash",
		"pkg:generic/tool",
		"",
		"",
	}

	for _, filename := range []string{"identity-test.manifest", "identity-test.lock"} {
		t.Run(filename, func(t *testing.T) {
			result, err := Parse(filename, nil)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(result.Declarations) != len(want) {
				t.Fatalf("got %d declarations, want %d", len(result.Declarations), len(want))
			}
			for i, declaration := range result.Declarations {
				if declaration.PURL != want[i] {
					t.Errorf("%s PURL = %q, want %q", declaration.Name, declaration.PURL, want[i])
				}
				if declaration.Ecosystem != identityTestEcosystems[i] {
					t.Errorf("%s Ecosystem = %q, want %q", declaration.Name, declaration.Ecosystem, identityTestEcosystems[i])
				}
			}
		})
	}
}

func TestManifestDependencyAndDeclarationIdentitiesMatch(t *testing.T) {
	// Without a parser-supplied PURL, a manifest dependency and its declaration
	// resolve to the same versionless identity.
	result, err := Parse("identity-test.manifest", nil)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	for _, i := range []int{0, 1, 3, 4} {
		dep, declaration := result.Dependencies[i], result.Declarations[i]
		if dep.PURL != declaration.PURL {
			t.Errorf("%s dependency PURL %q != declaration PURL %q", dep.Name, dep.PURL, declaration.PURL)
		}
		if dep.Ecosystem != declaration.Ecosystem {
			t.Errorf("%s dependency Ecosystem %q != declaration Ecosystem %q", dep.Name, dep.Ecosystem, declaration.Ecosystem)
		}
	}
}

func TestParseSetsFileEcosystemWhenParserDoesNotOptIn(t *testing.T) {
	for _, path := range []string{"testdata/npm/package.json", "testdata/npm/package-lock.json"} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}
			result, err := Parse(filepath.Base(path), content)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if len(result.Dependencies) == 0 {
				t.Fatal("expected dependencies")
			}
			for _, dep := range result.Dependencies {
				if dep.Ecosystem != "npm" {
					t.Errorf("%s Ecosystem = %q, want npm", dep.Name, dep.Ecosystem)
				}
			}
			for _, declaration := range result.Declarations {
				if declaration.Ecosystem != "npm" {
					t.Errorf("%s declaration Ecosystem = %q, want npm", declaration.Name, declaration.Ecosystem)
				}
			}
		})
	}
}

func TestDenoNPMImportPURLs(t *testing.T) {
	// https://github.com/git-pkgs/manifests/issues/111
	tests := []struct {
		path string
		want map[string]string
	}{
		{"testdata/npm/deno.json", map[string]string{
			"chalk":     "pkg:npm/chalk",
			"lodash":    "pkg:npm/lodash",
			"@std/path": "pkg:deno/%40std%2Fpath",
			"@std/fs":   "pkg:deno/%40std%2Ffs",
		}},
		{"testdata/npm/deno.lock", map[string]string{
			"chalk":     "pkg:npm/chalk@5.3.0",
			"lodash":    "pkg:npm/lodash@4.17.21",
			"@std/path": "pkg:deno/%40std%2Fpath@1.0.6",
			"@std/fs":   "pkg:deno/%40std%2Ffs@1.0.3",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			content, err := os.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}
			result, err := Parse(filepath.Base(tt.path), content)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if result.Ecosystem != "deno" {
				t.Errorf("Ecosystem = %q, want deno", result.Ecosystem)
			}
			if len(result.Dependencies) != len(tt.want) {
				t.Fatalf("got %d dependencies, want %d", len(result.Dependencies), len(tt.want))
			}
			for _, dep := range result.Dependencies {
				if want := tt.want[dep.Name]; dep.PURL != want {
					t.Errorf("%s PURL = %q, want %q", dep.Name, dep.PURL, want)
				}
				wantEcosystem := "deno"
				if dep.Name == "chalk" || dep.Name == "lodash" {
					wantEcosystem = "npm"
				}
				if dep.Ecosystem != wantEcosystem {
					t.Errorf("%s Ecosystem = %q, want %q", dep.Name, dep.Ecosystem, wantEcosystem)
				}
			}
		})
	}
}

func TestDenoImportSources(t *testing.T) {
	// https://github.com/git-pkgs/manifests/issues/116
	content := []byte(`{"imports": {
  "chalk": "npm:chalk@5.3.0",
  "@std/fs": "jsr:@std/fs",
  "$utils/": "./src/utils/",
  "std/": "http://deno.land/std@0.200.0/",
  "react": "https://esm.sh/react@18",
  "local": "file:///tmp/mod.ts",
  "fs": "node:fs",
  "inline": "data:text/javascript,export default 1",
  "bare": "lodash"
}}`)

	want := map[string]struct {
		purl, ecosystem string
		source          Source
	}{
		"chalk":   {"pkg:npm/chalk", "npm", Source{}},
		"@std/fs": {"pkg:deno/%40std%2Ffs", "deno", Source{}},
		"$utils/": {"", "", Source{Kind: SourcePath, Value: "./src/utils/"}},
		"std/":    {"", "", Source{Kind: SourceURL, Value: "http://deno.land/std@0.200.0/"}},
		"react":   {"", "", Source{Kind: SourceURL, Value: "https://esm.sh/react@18"}},
		"local":   {"", "", Source{Kind: SourcePath, Value: "file:///tmp/mod.ts"}},
	}

	result, err := Parse("deno.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(result.Dependencies) != len(want) {
		t.Fatalf("got %d dependencies, want %d: %+v", len(result.Dependencies), len(want), result.Dependencies)
	}
	for _, dep := range result.Dependencies {
		w, ok := want[dep.Name]
		if !ok {
			t.Errorf("unexpected dependency %q", dep.Name)
			continue
		}
		if dep.PURL != w.purl {
			t.Errorf("%s PURL = %q, want %q", dep.Name, dep.PURL, w.purl)
		}
		if dep.Ecosystem != w.ecosystem {
			t.Errorf("%s Ecosystem = %q, want %q", dep.Name, dep.Ecosystem, w.ecosystem)
		}
		if dep.Source != w.source {
			t.Errorf("%s Source = %+v, want %+v", dep.Name, dep.Source, w.source)
		}
		if dep.NoPURL != (w.purl == "") {
			t.Errorf("%s NoPURL = %v, want %v", dep.Name, dep.NoPURL, w.purl == "")
		}
	}
}

func TestRenvLockSources(t *testing.T) {
	// https://github.com/git-pkgs/manifests/issues/123
	content := []byte(`{
  "R": {
    "Version": "4.5.0",
    "Repositories": [
      {"Name": "CRAN", "URL": "https://cloud.r-project.org"},
      {"Name": "internal", "URL": "https://r.example.com"}
    ]
  },
  "Packages": {
    "BiocGenerics": {"Package": "BiocGenerics", "Version": "0.58.1", "Source": "Bioconductor", "Repository": "Bioconductor"},
    "survival": {"Package": "survival", "Version": "2.34", "Source": "Repository", "Repository": "CRAN", "Hash": "abc123"},
    "rspmpkg": {"Package": "rspmpkg", "Version": "1.2.0", "Source": "Repository", "Repository": "RSPM"},
    "ppmpkg": {"Package": "ppmpkg", "Version": "1.3.0", "Source": "Repository", "Repository": "PPM"},
    "p3mpkg": {"Package": "p3mpkg", "Version": "1.4.0", "Source": "Repository", "Repository": "P3M"},
    "IRanges": {"Package": "IRanges", "Version": "2.40.0", "Source": "Repository", "Repository": "BioCsoft"},
    "S4Vectors": {"Package": "S4Vectors", "Version": "0.44.0", "Source": "Bioconductor"},
    "legacy": {"Package": "legacy", "Version": "1.0.0"},
    "internalpkg": {"Package": "internalpkg", "Version": "0.2.0", "Source": "Repository", "Repository": "internal"},
    "universe": {"Package": "universe", "Version": "0.3.0", "Source": "Repository", "Repository": "https://ropensci.r-universe.dev"},
    "mystery": {"Package": "mystery", "Version": "0.4.0", "Source": "Repository", "Repository": "unlisted"},
    "ghpkg": {"Package": "ghpkg", "Version": "0.5.0", "Source": "GitHub", "RemoteType": "github", "RemoteHost": "api.github.com", "RemoteUsername": "acme", "RemoteRepo": "ghpkg", "RemoteRef": "main", "RemoteSha": "0123abc", "RemoteSubdir": "pkg"},
    "glpkg": {"Package": "glpkg", "Version": "0.6.0", "Source": "GitLab", "RemoteUsername": "acme", "RemoteRepo": "glpkg", "RemoteRef": "v0.6.0"},
    "gitpkg": {"Package": "gitpkg", "Version": "0.7.0", "Source": "git", "RemoteUrl": "https://git.example.com/gitpkg.git", "RemoteSha": "4567def"},
    "urlpkg": {"Package": "urlpkg", "Version": "0.8.0", "Source": "URL", "RemoteUrl": "https://example.com/urlpkg_0.8.0.tar.gz"},
    "localpkg": {"Package": "localpkg", "Version": "0.9.0", "Source": "Local", "RemoteUrl": "~/src/localpkg"},
    "unknownpkg": {"Package": "unknownpkg", "Version": "1.1.0", "Source": "unknown"}
  }
}`)

	want := map[string]struct {
		purl, ecosystem string
		source          Source
	}{
		"BiocGenerics": {"pkg:bioconductor/BiocGenerics@0.58.1", "bioconductor", Source{Kind: SourceRegistry, Value: "Bioconductor"}},
		"survival":     {"pkg:cran/survival@2.34", "cran", Source{Kind: SourceRegistry, Value: "CRAN"}},
		"rspmpkg":      {"pkg:cran/rspmpkg@1.2.0", "cran", Source{Kind: SourceRegistry, Value: "RSPM"}},
		"ppmpkg":       {"pkg:cran/ppmpkg@1.3.0", "cran", Source{Kind: SourceRegistry, Value: "PPM"}},
		"p3mpkg":       {"pkg:cran/p3mpkg@1.4.0", "cran", Source{Kind: SourceRegistry, Value: "P3M"}},
		"IRanges":      {"pkg:bioconductor/IRanges@2.40.0", "bioconductor", Source{Kind: SourceRegistry, Value: "BioCsoft"}},
		"S4Vectors":    {"pkg:bioconductor/S4Vectors@0.44.0", "bioconductor", Source{Kind: SourceRegistry, Value: "Bioconductor"}},
		"legacy":       {"pkg:cran/legacy@1.0.0", "cran", Source{}},
		"internalpkg":  {"pkg:cran/internalpkg@0.2.0?repository_url=https:%2F%2Fr.example.com", "cran", Source{Kind: SourceRegistry, Value: "internal"}},
		"universe":     {"pkg:cran/universe@0.3.0?repository_url=https:%2F%2Fropensci.r-universe.dev", "cran", Source{Kind: SourceRegistry, Value: "https://ropensci.r-universe.dev"}},
		"mystery":      {"", "", Source{Kind: SourceRegistry, Value: "unlisted"}},
		"ghpkg":        {"", "", Source{Kind: SourceGitHub, Value: "acme/ghpkg", Ref: "0123abc", Rel: "pkg"}},
		"glpkg":        {"", "", Source{Kind: SourceGit, Value: "gitlab::acme/glpkg", Ref: "v0.6.0"}},
		"gitpkg":       {"", "", Source{Kind: SourceGit, Value: "https://git.example.com/gitpkg.git", Ref: "4567def"}},
		"urlpkg":       {"", "", Source{Kind: SourceURL, Value: "https://example.com/urlpkg_0.8.0.tar.gz"}},
		"localpkg":     {"", "", Source{Kind: SourcePath, Value: "~/src/localpkg"}},
		"unknownpkg":   {"", "", Source{}},
	}

	result, err := Parse("renv.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(result.Dependencies) != len(want) {
		t.Fatalf("got %d dependencies, want %d: %+v", len(result.Dependencies), len(want), result.Dependencies)
	}
	for _, dep := range result.Dependencies {
		w, ok := want[dep.Name]
		if !ok {
			t.Errorf("unexpected dependency %q", dep.Name)
			continue
		}
		if dep.PURL != w.purl {
			t.Errorf("%s PURL = %q, want %q", dep.Name, dep.PURL, w.purl)
		}
		if dep.Ecosystem != w.ecosystem {
			t.Errorf("%s Ecosystem = %q, want %q", dep.Name, dep.Ecosystem, w.ecosystem)
		}
		if dep.Source != w.source {
			t.Errorf("%s Source = %+v, want %+v", dep.Name, dep.Source, w.source)
		}
		if dep.NoPURL != (w.purl == "") {
			t.Errorf("%s NoPURL = %v, want %v", dep.Name, dep.NoPURL, w.purl == "")
		}
		if dep.Direct {
			t.Errorf("%s Direct = true, want false", dep.Name)
		}
		if dep.Version == "" {
			t.Errorf("%s Version is empty", dep.Name)
		}
		if dep.Name == "survival" && dep.Integrity != "md5-abc123" {
			t.Errorf("survival Integrity = %q, want %q", dep.Integrity, "md5-abc123")
		}
	}
}
