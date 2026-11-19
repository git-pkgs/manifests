package manifests

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRegistryConfig(t *testing.T) {
	tests := []struct {
		fixture   string
		filename  string
		ecosystem string
		kind      Kind
		want      []Registry
	}{
		{
			fixture: "testdata/npm/registry-config/.npmrc", filename: ".npmrc", ecosystem: "npm", kind: Config,
			want: []Registry{
				{URL: "https://nexus.internal/repository/npm-group/", Replaces: true},
				{URL: "https://nexus.internal/repository/acme/", Scope: "@acme", Replaces: true},
			},
		},
		{
			fixture: "testdata/npm/registry-config/.yarnrc.yml", filename: ".yarnrc.yml", ecosystem: "npm", kind: Config,
			want: []Registry{
				{URL: "https://nexus.internal/repository/npm-group/", Replaces: true},
				{URL: "https://nexus.internal/repository/acme/", Scope: "@acme", Replaces: true},
				{URL: "https://npm.tools.example/", Scope: "@tools", Replaces: true},
			},
		},
		{
			fixture: "testdata/pypi/registry-config/pip.conf", filename: "pip.conf", ecosystem: "pypi", kind: Config,
			want: []Registry{
				{URL: "https://nexus.internal/repository/pypi-group/simple", Replaces: true},
				{URL: "https://download.pytorch.org/whl/cpu"},
				{URL: "https://pypi.acme.example/simple"},
				{URL: "https://pypi.tools.example/simple"},
			},
		},
		{
			fixture: "testdata/pypi/registry-config/pip.ini", filename: "pip.ini", ecosystem: "pypi", kind: Config,
			want: []Registry{{URL: "https://nexus.internal/pypi/simple", Replaces: true}},
		},
		{
			fixture: "testdata/cargo/registry-config/.cargo/config.toml", filename: "project/.cargo/config.toml", ecosystem: "cargo", kind: Config,
			want: []Registry{{URL: "sparse+https://cargo.internal/api/v1/crates/", Replaces: true}},
		},
		{
			fixture: "testdata/cargo/registry-alternative/.cargo/config.toml", filename: ".cargo/config.toml", ecosystem: "cargo", kind: Config,
			want: []Registry{{URL: "sparse+https://cargo.internal/index/", Replaces: true}},
		},
		{
			fixture: "testdata/cargo/registry-config/.cargo/config", filename: ".cargo/config", ecosystem: "cargo", kind: Config,
			want: []Registry{{URL: "https://github.com/rust-lang/crates.io-index-mirror", Replaces: true}},
		},
		{
			fixture: "testdata/gem/registry-config/.bundle/config", filename: ".bundle/config", ecosystem: "gem", kind: Config,
			want: []Registry{
				{URL: "https://gems.internal/rubygems/", Scope: "https://rubygems.org/", Replaces: true},
				{URL: "https://gems-mirror.internal/", Scope: "https://gems.my-company.example/", Replaces: true},
				{URL: "https://everything.internal/", Replaces: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			content, err := os.ReadFile(tt.fixture)
			if err != nil {
				t.Fatal(err)
			}
			result, err := Parse(tt.filename, content)
			if err != nil {
				t.Fatal(err)
			}
			if result.Ecosystem != tt.ecosystem || result.Kind != tt.kind {
				t.Errorf("identified as %s %s, want %s %s", result.Ecosystem, result.Kind, tt.ecosystem, tt.kind)
			}
			for i := range tt.want {
				tt.want[i].File = tt.filename
			}
			if !reflect.DeepEqual(result.Registries, tt.want) {
				t.Errorf("registries =\n%+v\nwant\n%+v", result.Registries, tt.want)
			}
			if len(result.Dependencies) != 0 {
				t.Errorf("dependencies = %+v, want none", result.Dependencies)
			}
			assertNoCredentials(t, result)
		})
	}
}

// assertNoCredentials fails if any fixture secret reaches the result.
func assertNoCredentials(t *testing.T, result *ParseResult) {
	t.Helper()
	dump := fmt.Sprintf("%+v", *result)
	for _, secret := range []string{"s3cret", "SECRET", "c2Vj", "deploy:", "ci:", "token:"} {
		if strings.Contains(dump, secret) {
			t.Errorf("result contains %q: %s", secret, dump)
		}
	}
}

func TestNpmRCRegistryKeepsProvenance(t *testing.T) {
	result, err := Parse(".npmrc", []byte("provenance=true\n@acme:registry=https://npm.acme.example/\n"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Publication.Provenance == nil || !*result.Publication.Provenance || len(result.Registries) != 1 {
		t.Errorf("result = %+v", result)
	}
}

func TestRegistryConfigIdentify(t *testing.T) {
	for filename, want := range map[string]string{
		".npmrc":                                "npm",
		".yarnrc.yml":                           "npm",
		"pip.conf":                              "pypi",
		"config/pip.ini":                        "pypi",
		".cargo/config":                         "cargo",
		"crates/app/.cargo/config.toml":         "cargo",
		filepath.Join("a", ".bundle", "config"): "gem",
	} {
		ecosystem, kind, ok := Identify(filename)
		if !ok || ecosystem != want || kind != Config {
			t.Errorf("Identify(%q) = %q %q %v, want %q config", filename, ecosystem, kind, ok, want)
		}
	}
	for _, filename := range []string{"config", "config.toml", "x.cargo/config.toml", ".bundle/config.yml", "cargo/config"} {
		if _, kind, ok := Identify(filename); ok && kind == Config {
			t.Errorf("Identify(%q) matched a config parser", filename)
		}
	}
}

func TestRegistryURLStripsCredentials(t *testing.T) {
	for raw, want := range map[string]string{
		"https://user:pass@host/path":      "https://host/path",
		"https://user:p@ss@host/":          "https://host/",
		"sparse+https://token@host/index/": "sparse+https://host/index/",
		"//user:pass@host/":                "//host/",
		"https://host/path?x=a@b":          "https://host/path?x=a@b",
		" https://host/ ":                  "https://host/",
		"${NPM_REGISTRY}":                  "${NPM_REGISTRY}",
	} {
		result, err := Parse("pip.conf", []byte("[global]\nindex-url = "+raw+"\n"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Registries) != 1 || result.Registries[0].URL != want {
			t.Errorf("%q: registries = %+v, want %q", raw, result.Registries, want)
		}
	}
}

func TestDiscoverManifestsConfigFiles(t *testing.T) {
	reader := mapFSReader(map[string]string{
		".npmrc":                    "registry=https://npm.internal/\n",
		".yarnrc.yml":               "npmRegistryServer: https://npm.internal/\n",
		"pip.conf":                  "[global]\nindex-url = https://pypi.internal/simple\n",
		"pip.ini":                   "[global]\nindex-url = https://pypi.internal/simple\n",
		".cargo/config.toml":        "[source.crates-io]\nreplace-with = \"corp\"\n",
		".cargo/config":             "[source.crates-io]\nreplace-with = \"corp\"\n",
		".bundle/config":            "BUNDLE_MIRROR__ALL: https://gems.internal/\n",
		"nested/.cargo/config.toml": "",
	})
	got, warnings := DiscoverManifests(reader)
	if len(warnings) != 0 {
		t.Fatalf("DiscoverManifests warnings: %v", warnings)
	}
	want := []DiscoveredManifest{
		{Path: ".bundle/config", Ecosystem: "gem", Kind: Config},
		{Path: ".cargo/config", Ecosystem: "cargo", Kind: Config},
		{Path: ".cargo/config.toml", Ecosystem: "cargo", Kind: Config},
		{Path: ".npmrc", Ecosystem: "npm", Kind: Config},
		{Path: ".yarnrc.yml", Ecosystem: "npm", Kind: Config},
		{Path: "pip.conf", Ecosystem: "pypi", Kind: Config},
		{Path: "pip.ini", Ecosystem: "pypi", Kind: Config},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("DiscoverManifests() =\n%+v\nwant\n%+v", got, want)
	}
}
