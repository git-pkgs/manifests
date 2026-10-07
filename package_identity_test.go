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
			{Name: "tool", Version: "2.0.0", PURL: "pkg:generic/tool@2.0.0"},
			{Name: "lib.jar", Version: "3.0.0", NoPURL: true, Source: jar},
			{Name: "ignored", Version: "4.0.0", NoPURL: true, PURL: "pkg:generic/ignored@4.0.0"},
		},
		Declarations: []core.Declaration{
			{Name: "org.example:fallback", Version: "1.0.0", Location: "fallback"},
			{Name: "lodash", Version: "4.17.21", Ecosystem: "npm", Location: "lodash"},
			{Name: "tool", Version: "2.0.0", PURL: "pkg:generic/tool", Location: "tool"},
			{Name: "lib.jar", Version: "3.0.0", NoPURL: true, Source: jar, Location: "lib.jar"},
			{Name: "ignored", Version: "4.0.0", NoPURL: true, PURL: "pkg:generic/ignored", Location: "ignored"},
		},
	}, nil
}

func init() {
	core.Register("maven", core.Manifest, identityTestParser{}, core.ExactMatch("identity-test.manifest"))
	core.Register("maven", core.Lockfile, identityTestParser{}, core.ExactMatch("identity-test.lock"))
}

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
			}
		})
	}
}
