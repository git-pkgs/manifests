package manifests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/purl"
)

func TestParseAllEcosystems(t *testing.T) {
	// Test that we can parse files from each ecosystem
	testCases := []struct {
		name      string
		path      string
		ecosystem string
		kind      Kind
	}{
		{"npm package.json", "testdata/npm/package.json", "npm", Manifest},
		{"npm package-lock.json", "testdata/npm/package-lock.json", "npm", Lockfile},
		{"gem Gemfile", "testdata/gem/Gemfile", "gem", Manifest},
		{"gem Gemfile.lock", "testdata/gem/Gemfile.lock", "gem", Lockfile},
		{"cargo Cargo.toml", "testdata/cargo/Cargo.toml", "cargo", Manifest},
		{"cargo Cargo.lock", "testdata/cargo/Cargo.lock", "cargo", Lockfile},
		{"golang go.mod", "testdata/golang/go.mod", "golang", Manifest},
		{"golang go.sum", "testdata/golang/go.sum", "golang", Lockfile},
		{"pypi requirements.txt", "testdata/pypi/requirements.txt", "pypi", Manifest},
		{"maven pom.xml", "testdata/maven/pom.xml", "maven", Manifest},
		{"composer composer.json", "testdata/composer/composer.json", "composer", Manifest},
		{"composer composer.lock", "testdata/composer/composer.lock", "composer", Lockfile},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := os.ReadFile(tc.path)
			if err != nil {
				t.Skipf("fixture not found: %v", err)
			}

			filename := filepath.Base(tc.path)
			result, err := Parse(filename, content)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result.Ecosystem != tc.ecosystem {
				t.Errorf("Ecosystem = %q, want %q", result.Ecosystem, tc.ecosystem)
			}
			if result.Kind != tc.kind {
				t.Errorf("Kind = %q, want %q", result.Kind, tc.kind)
			}
			if len(result.Dependencies) == 0 {
				t.Error("expected dependencies, got none")
			}
		})
	}
}

func TestIdentifyFiles(t *testing.T) {
	testCases := []struct {
		filename  string
		ecosystem string
		kind      Kind
		ok        bool
	}{
		// npm
		{"package.json", "npm", Manifest, true},
		{"package-lock.json", "npm", Lockfile, true},
		{"npm-shrinkwrap.json", "npm", Lockfile, true},
		{"yarn.lock", "npm", Lockfile, true},
		{"pnpm-lock.yaml", "npm", Lockfile, true},

		// gem
		{"Gemfile", "gem", Manifest, true},
		{"gems.rb", "gem", Manifest, true},
		{"Gemfile.lock", "gem", Lockfile, true},
		{"foo.gemspec", "gem", Manifest, true},

		// cargo
		{"Cargo.toml", "cargo", Manifest, true},
		{"Cargo.lock", "cargo", Lockfile, true},

		// golang
		{"go.mod", "golang", Manifest, true},
		{"go.sum", "golang", Lockfile, true},

		// pypi
		{"requirements.txt", "pypi", Manifest, true},
		{"Pipfile", "pypi", Manifest, true},
		{"Pipfile.lock", "pypi", Lockfile, true},
		{"pyproject.toml", "pypi", Manifest, true},
		{"poetry.lock", "pypi", Lockfile, true},

		// maven
		{"pom.xml", "maven", Manifest, true},

		// composer
		{"composer.json", "composer", Manifest, true},
		{"composer.lock", "composer", Lockfile, true},

		// docker
		{"Dockerfile", "docker", Manifest, true},
		{"docker-compose.yml", "docker", Manifest, true},

		// unknown
		{"unknown.txt", "", "", false},
		{"random.file", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			ecosystem, kind, ok := Identify(tc.filename)
			if ok != tc.ok {
				t.Errorf("ok = %v, want %v", ok, tc.ok)
			}
			if ecosystem != tc.ecosystem {
				t.Errorf("ecosystem = %q, want %q", ecosystem, tc.ecosystem)
			}
			if kind != tc.kind {
				t.Errorf("kind = %q, want %q", kind, tc.kind)
			}
		})
	}
}

func TestDependencyScopes(t *testing.T) {
	// Test that dependencies have correct scopes
	content, err := os.ReadFile("testdata/npm/package.json")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	result, err := Parse("package.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	deps := make(map[string]Dependency)
	for _, d := range result.Dependencies {
		deps[d.Name] = d
	}

	// Check runtime dependency
	if babel, ok := deps["babel"]; ok {
		if babel.Scope != Runtime {
			t.Errorf("babel scope = %q, want %q", babel.Scope, Runtime)
		}
	}

	// Check dev dependency
	if mocha, ok := deps["mocha"]; ok {
		if mocha.Scope != Development {
			t.Errorf("mocha scope = %q, want %q", mocha.Scope, Development)
		}
	}
}

func TestIntegrity(t *testing.T) {
	// Test that lockfiles have integrity hashes
	content, err := os.ReadFile("testdata/npm/package-lock.json")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	result, err := Parse("package-lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	hasIntegrity := false
	for _, d := range result.Dependencies {
		if d.Integrity != "" {
			hasIntegrity = true
			break
		}
	}

	if !hasIntegrity {
		t.Error("expected at least one dependency with integrity hash")
	}
}

func TestPURL(t *testing.T) {
	// Test PURL generation
	content, err := os.ReadFile("testdata/npm/package-lock.json")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	result, err := Parse("package-lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for _, d := range result.Dependencies {
		if d.Name == "express" {
			if d.PURL != "pkg:npm/express@4.15.3" {
				t.Errorf("express PURL = %q, want %q", d.PURL, "pkg:npm/express@4.15.3")
			}
			return
		}
	}
	t.Error("express dependency not found")
}

func TestRegistryURLNotIncludedForDefaultRegistry(t *testing.T) {
	// Test that default registry URLs don't add repository_url qualifier
	testCases := []struct {
		name        string
		content     string
		filename    string
		wantInPURL  bool
	}{
		{
			name: "npm default registry",
			content: `{
				"lockfileVersion": 3,
				"packages": {
					"node_modules/lodash": {
						"version": "4.17.21",
						"resolved": "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz"
					}
				}
			}`,
			filename:   "package-lock.json",
			wantInPURL: false,
		},
		{
			name: "npm yarn registry (non-canonical)",
			content: `{
				"lockfileVersion": 3,
				"packages": {
					"node_modules/lodash": {
						"version": "4.17.21",
						"resolved": "https://registry.yarnpkg.com/lodash/-/lodash-4.17.21.tgz"
					}
				}
			}`,
			filename:   "package-lock.json",
			wantInPURL: true, // yarn is not the canonical npm registry
		},
		{
			name: "npm private registry",
			content: `{
				"lockfileVersion": 3,
				"packages": {
					"node_modules/lodash": {
						"version": "4.17.21",
						"resolved": "https://npm.mycompany.com/lodash/-/lodash-4.17.21.tgz"
					}
				}
			}`,
			filename:   "package-lock.json",
			wantInPURL: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parse(tc.filename, []byte(tc.content))
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(result.Dependencies) == 0 {
				t.Fatal("expected dependencies")
			}

			dep := result.Dependencies[0]
			hasRepositoryURL := strings.Contains(dep.PURL, "repository_url=")

			if hasRepositoryURL != tc.wantInPURL {
				t.Errorf("PURL = %q, wantInPURL = %v", dep.PURL, tc.wantInPURL)
			}
		})
	}
}

func TestRegistryURLQualifier(t *testing.T) {
	// Test that private registry URLs are encoded in PURL
	content := `{
		"lockfileVersion": 3,
		"packages": {
			"node_modules/@mycompany/sdk": {
				"version": "1.0.0",
				"resolved": "https://npm.mycompany.com/@mycompany/sdk/-/sdk-1.0.0.tgz"
			}
		}
	}`

	result, err := Parse("package-lock.json", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Dependencies) == 0 {
		t.Fatal("expected dependencies")
	}

	dep := result.Dependencies[0]

	// Should have repository_url qualifier
	if !strings.Contains(dep.PURL, "repository_url=") {
		t.Errorf("expected repository_url qualifier in PURL, got %q", dep.PURL)
	}

	// Should have the private registry URL (URL-encoded)
	if !strings.Contains(dep.PURL, "npm.mycompany.com") {
		t.Errorf("expected private registry URL in PURL, got %q", dep.PURL)
	}
}

func TestIsNonDefaultRegistry(t *testing.T) {
	// Tests use purl.IsNonDefaultRegistry which checks against types.json default_registry.
	// Only the canonical registry is considered "default" - mirrors and alternatives are non-default.
	testCases := []struct {
		ecosystem   string
		registryURL string
		want        bool
	}{
		{"npm", "https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz", false},
		{"npm", "https://registry.yarnpkg.com/lodash/-/lodash-4.17.21.tgz", true}, // yarn is not the canonical npm registry
		{"npm", "https://npm.mycompany.com/lodash/-/lodash-4.17.21.tgz", true},
		{"npm", "", false},
		{"pypi", "https://pypi.org/packages/foo.whl", false},
		{"pypi", "https://files.pythonhosted.org/packages/foo.whl", true}, // pythonhosted is a CDN, not the canonical registry
		{"pypi", "https://private.pypi.company.com/foo.whl", true},
		{"cargo", "https://crates.io/api/v1/crates/foo", false},
		{"cargo", "https://index.crates.io/foo", false}, // subdomain of crates.io
		{"cargo", "https://private.cargo.company.com/foo", true},
		{"gem", "https://rubygems.org/gems/foo.gem", false},
		{"gem", "https://private.gems.company.com/foo.gem", true},
		{"composer", "https://packagist.org/packages/foo", false},
		{"composer", "https://repo.packagist.org/packages/foo", false}, // subdomain of packagist.org
		{"composer", "https://private.packagist.company.com/foo", true},
		{"unknown", "https://example.com/foo", true},
	}

	for _, tc := range testCases {
		t.Run(tc.ecosystem+"_"+tc.registryURL, func(t *testing.T) {
			purlType := purl.EcosystemToPURLType(tc.ecosystem)
			got := purl.IsNonDefaultRegistry(purlType, tc.registryURL)
			if got != tc.want {
				t.Errorf("purl.IsNonDefaultRegistry(%q, %q) = %v, want %v", purlType, tc.registryURL, got, tc.want)
			}
		})
	}
}

func TestGemRegistryURL(t *testing.T) {
	content := `GEM
  remote: https://rubygems.org/
  specs:
    rails (7.0.0)

GEM
  remote: https://gems.mycompany.com/
  specs:
    private-gem (1.0.0)

PLATFORMS
  ruby

DEPENDENCIES
  rails
  private-gem
`

	result, err := Parse("Gemfile.lock", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	deps := make(map[string]Dependency)
	for _, d := range result.Dependencies {
		deps[d.Name] = d
	}

	// Rails from default registry should not have repository_url
	if rails, ok := deps["rails"]; ok {
		if strings.Contains(rails.PURL, "repository_url=") {
			t.Errorf("rails PURL should not have repository_url, got %q", rails.PURL)
		}
	} else {
		t.Error("rails dependency not found")
	}

	// Private gem should have repository_url
	if privateGem, ok := deps["private-gem"]; ok {
		if !strings.Contains(privateGem.PURL, "repository_url=") {
			t.Errorf("private-gem PURL should have repository_url, got %q", privateGem.PURL)
		}
	} else {
		t.Error("private-gem dependency not found")
	}
}

func TestPipfileLockRegistryURL(t *testing.T) {
	content := `{
    "_meta": {
        "sources": [
            {"name": "pypi", "url": "https://pypi.org/simple"},
            {"name": "private", "url": "https://private.pypi.company.com/simple"}
        ]
    },
    "default": {
        "requests": {
            "version": "==2.31.0",
            "index": "pypi"
        },
        "private-pkg": {
            "version": "==1.0.0",
            "index": "private"
        },
        "direct-file": {
            "file": "https://github.com/user/repo/releases/download/v1.0.0/pkg.whl"
        }
    },
    "develop": {}
}`

	result, err := Parse("Pipfile.lock", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	deps := make(map[string]Dependency)
	for _, d := range result.Dependencies {
		deps[d.Name] = d
	}

	// requests from pypi should not have repository_url
	if requests, ok := deps["requests"]; ok {
		if strings.Contains(requests.PURL, "repository_url=") {
			t.Errorf("requests PURL should not have repository_url, got %q", requests.PURL)
		}
	}

	// private-pkg should have repository_url
	if privatePkg, ok := deps["private-pkg"]; ok {
		if !strings.Contains(privatePkg.PURL, "repository_url=") {
			t.Errorf("private-pkg PURL should have repository_url, got %q", privatePkg.PURL)
		}
	}

	// direct-file should have repository_url (github is not a default pypi registry)
	if directFile, ok := deps["direct-file"]; ok {
		if !strings.Contains(directFile.PURL, "repository_url=") {
			t.Errorf("direct-file PURL should have repository_url, got %q", directFile.PURL)
		}
	}
}
