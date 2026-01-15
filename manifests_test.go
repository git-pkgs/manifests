package manifests

import (
	"os"
	"path/filepath"
	"testing"
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
