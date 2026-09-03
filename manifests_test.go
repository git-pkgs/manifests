package manifests

import (
	"os"
	"path/filepath"
	"slices"
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
		{"golang go.sum", "testdata/golang/go.sum", "golang", Supplement},
		{"pypi requirements.txt", "testdata/pypi/requirements.txt", "pypi", Manifest},
		{"maven pom.xml", "testdata/maven/pom.xml", "maven", Manifest},
		{"nuget central packages", "testdata/nuget/Directory.Packages.props", "nuget", Manifest},
		{"chef metadata.rb", "testdata/chef/metadata.rb", "chef", Manifest},
		{"chef metadata.json", "testdata/chef/metadata.json", "chef", Manifest},
		{"chef Berksfile", "testdata/chef/Berksfile", "chef", Manifest},
		{"composer composer.json", "testdata/composer/composer.json", "composer", Manifest},
		{"composer composer.lock", "testdata/composer/composer.lock", "composer", Lockfile},
		{"helm Chart.yaml", "testdata/helm/Chart.yaml", "helm", Manifest},
		{"helm Chart.lock", "testdata/helm/Chart.lock", "helm", Lockfile},
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

func TestEcosystems(t *testing.T) {
	got := Ecosystems()

	if len(got) == 0 {
		t.Fatal("Ecosystems() returned empty list")
	}

	seen := make(map[string]bool)
	for _, e := range got {
		if seen[e] {
			t.Errorf("duplicate ecosystem %q", e)
		}
		seen[e] = true
	}

	for _, want := range []string{"npm", "gem", "cargo", "golang", "pypi", "maven", "chef", "helm"} {
		if !slices.Contains(got, want) {
			t.Errorf("Ecosystems() missing %q", want)
		}
	}
}

func TestChefManifestRegistrationAndCandidatePURLs(t *testing.T) {
	t.Parallel()

	for _, filename := range []string{"metadata.rb", "metadata.json", "Berksfile"} {
		ecosystem, kind, ok := Identify("cookbooks/example/" + filename)
		if !ok || ecosystem != "chef" || kind != Manifest {
			t.Errorf("Identify(%q) = %q, %q, %v; want chef manifest", filename, ecosystem, kind, ok)
		}
	}

	content, err := os.ReadFile("testdata/chef/Berksfile")
	if err != nil {
		t.Fatal(err)
	}
	result, err := Parse("Berksfile", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sources) != 2 || result.Sources[0].Value != "https://supermarket.chef.io" ||
		result.Sources[1].Value != "https://supermarket.example.test" {
		t.Errorf("sources = %+v, want ordered Berksfile sources", result.Sources)
	}
	for _, dependency := range result.Dependencies {
		if dependency.PURL != "" {
			t.Errorf("Chef candidate PURL = %q, want empty", dependency.PURL)
		}
	}
	for _, declaration := range result.Declarations {
		if declaration.PURL != "" {
			t.Errorf("Chef candidate declaration PURL = %q, want empty", declaration.PURL)
		}
	}
}

func TestBerksfileMetadataDirectiveDoesNotReadFSRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	metadata := []byte("name 'must_not_be_loaded'\ndepends 'must_not_be_loaded'\n")
	if err := os.WriteFile(filepath.Join(root, "metadata.rb"), metadata, 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := Parse("Berksfile", []byte("metadata\n"), Options{FSRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "" || len(result.Dependencies) != 0 || len(result.Declarations) != 0 {
		t.Errorf("bare metadata loaded adjacent file: %+v", result)
	}
}

func TestChefParseRecoversAfterUnmatchedParenthesis(t *testing.T) {
	t.Parallel()

	content := []byte("cookbook(\"broken\"\ncookbook \"valid\", \"~> 1.0\"\n")
	result, err := Parse("Berksfile", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "valid" ||
		result.Dependencies[0].Version != "~> 1.0" {
		t.Errorf("dependencies = %+v, want recovered valid dependency", result.Dependencies)
	}
}

func TestChefParseTracksRubyBlockScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
	}{
		{
			name: "brace block parameters",
			content: `items.each { |item|
  cookbook "nested", path: "../nested"
}
cookbook "top_level", path: "../top_level"
`,
		},
		{
			name: "assigned conditional expression",
			content: `choice = if enabled
  cookbook "nested", path: "../nested"
end
cookbook "top_level", path: "../top_level"
`,
		},
		{
			name: "balanced one-line conditional",
			content: `if enabled; cookbook "same_line"; end
cookbook "top_level", path: "../top_level"
`,
		},
		{
			name: "loop keyword with optional do",
			content: `for dep in deps do
  cookbook "nested", path: "../nested"
end
cookbook "top_level", path: "../top_level"
`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result, err := Parse("Berksfile", []byte(test.content))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "top_level" {
				t.Errorf("dependencies = %+v, want only top_level", result.Dependencies)
			}
		})
	}
}

func TestMavenDeclarationPURLs(t *testing.T) {
	content := []byte(`<project>
  <parent>
    <groupId>org.example</groupId>
    <artifactId>parent</artifactId>
    <version>1.0.0</version>
  </parent>
  <artifactId>example</artifactId>
  <build>
    <plugins>
      <plugin>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>4.0.0</version>
      </plugin>
    </plugins>
  </build>
</project>`)

	result, err := Parse("pom.xml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := map[string]string{
		"parent/org.example:parent":                                    "pkg:maven/org.example/parent",
		"build/plugins/org.apache.maven.plugins:maven-compiler-plugin": "pkg:maven/org.apache.maven.plugins/maven-compiler-plugin",
	}
	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		wantPURL, ok := want[declaration.Location]
		if !ok {
			t.Errorf("unexpected declaration at %q: %+v", declaration.Location, declaration)
			continue
		}
		if declaration.PURL != wantPURL {
			t.Errorf("declaration at %q has PURL %q, want %q", declaration.Location, declaration.PURL, wantPURL)
		}
	}
}

func TestDeclarationPURLs(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		content     string
		location    string
		wantName    string
		wantVersion string
		wantPURL    string
	}{
		{
			name:        "npm alias",
			filename:    "package.json",
			content:     `{"dependencies":{"alias":"npm:@scope/actual-package@1.2.3"}}`,
			location:    "dependencies/alias",
			wantName:    "@scope/actual-package",
			wantVersion: "1.2.3",
			wantPURL:    "pkg:npm/%40scope/actual-package",
		},
		{
			name:        "canonical pypi name",
			filename:    "requirements.txt",
			content:     "Django_Rest.Framework==1.0\n",
			location:    "requirements/django-rest-framework",
			wantName:    "Django_Rest.Framework",
			wantVersion: "==1.0",
			wantPURL:    "pkg:pypi/django-rest.framework",
		},
		{
			name:        "github action subpath",
			filename:    ".github/workflows/ci.yml",
			content:     "jobs:\n  build:\n    steps:\n      - uses: actions/cache/restore@v4\n",
			location:    "jobs/build/steps/actions%2Fcache%2Frestore",
			wantName:    "actions/cache/restore",
			wantVersion: "v4",
			wantPURL:    "pkg:githubactions/actions/cache",
		},
		{
			name:        "cargo alias",
			filename:    "Cargo.toml",
			content:     "[dependencies]\nalias = { package = \"actual\", version = \"=1.0.0\" }\n",
			location:    "dependencies/alias",
			wantName:    "actual",
			wantVersion: "=1.0.0",
			wantPURL:    "pkg:cargo/actual",
		},
		{
			name:        "go requirement",
			filename:    "go.mod",
			content:     "module example.com/app\nrequire example.com/library v1.0.0\n",
			location:    "require/example.com%2Flibrary",
			wantName:    "example.com/library",
			wantVersion: "v1.0.0",
			wantPURL:    "pkg:golang/example.com/library",
		},
		{
			name:        "gleam dependency",
			filename:    "gleam.toml",
			content:     "[dependencies]\ngleam_stdlib = \"== 1.0.0\"\n",
			location:    "dependencies/gleam_stdlib",
			wantName:    "gleam_stdlib",
			wantVersion: "== 1.0.0",
			wantPURL:    "pkg:hex/gleam_stdlib",
		},
		{
			name:        "nuget central package",
			filename:    "Directory.Packages.props",
			content:     `<Project><ItemGroup><PackageVersion Include="example" Version="1.0.0" /></ItemGroup></Project>`,
			location:    "package-versions/example",
			wantName:    "example",
			wantVersion: "1.0.0",
			wantPURL:    "pkg:nuget/example",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse(test.filename, []byte(test.content))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(result.Declarations) != 1 {
				t.Fatalf("Declarations has %d entries, want 1: %+v", len(result.Declarations), result.Declarations)
			}
			declaration := result.Declarations[0]
			if declaration.Location != test.location || declaration.Name != test.wantName ||
				declaration.Version != test.wantVersion || declaration.PURL != test.wantPURL || !declaration.Direct {
				t.Errorf("Declaration = %+v, want location %q, name %q, version %q, PURL %q",
					declaration, test.location, test.wantName, test.wantVersion, test.wantPURL)
			}
		})
	}
}

func TestDeclarationParserPURLIsPreserved(t *testing.T) {
	declaration := Declaration{Name: "example", PURL: "pkg:npm/example"}

	if got, want := declarationPURL("deno", declaration), declaration.PURL; got != want {
		t.Fatalf("declarationPURL() = %q, want %q", got, want)
	}
}

func TestParseDeclaredLicenses(t *testing.T) {
	testCases := []struct {
		name        string
		filename    string
		content     string
		licenses    []string
		licenseFile string
	}{
		{
			name:     "npm scalar",
			filename: "package.json",
			content:  `{"name":"example","version":"1.0.0","license":"MIT"}`,
			licenses: []string{"MIT"},
		},
		{
			name:     "npm legacy object and array",
			filename: "package.json",
			content:  `{"license":{"type":"ISC","url":"https://example.test/ISC"},"licenses":[{"type":"MIT"},{"type":"Apache-2.0"}]}`,
			licenses: []string{"ISC", "MIT", "Apache-2.0"},
		},
		{
			name:        "cargo",
			filename:    "Cargo.toml",
			content:     "[package]\nname = \"example\"\nversion = \"1.0.0\"\nlicense = \"MIT OR Apache-2.0\"\nlicense-file = \"LICENSE.md\"\n",
			licenses:    []string{"MIT OR Apache-2.0"},
			licenseFile: "LICENSE.md",
		},
		{
			name:        "pyproject PEP 639",
			filename:    "pyproject.toml",
			content:     "[project]\nname = \"example\"\nversion = \"1.0.0\"\nlicense = \"MIT\"\nlicense-files = [\"LICENSE\", \"NOTICE\"]\n",
			licenses:    []string{"MIT"},
			licenseFile: "LICENSE",
		},
		{
			name:     "pyproject legacy metadata",
			filename: "pyproject.toml",
			content: `[project]
name = "example"
version = "1.0.0"
license = {text = "BSD-3-Clause"}
classifiers = ["Development Status :: 5 - Production/Stable", "License :: OSI Approved :: BSD License"]
`,
			licenses: []string{"BSD-3-Clause", "License :: OSI Approved :: BSD License"},
		},
		{
			name:        "setup cfg",
			filename:    "setup.cfg",
			content:     "[metadata]\nname = example\nversion = 1.0.0\nlicense = MIT\nlicense_files =\n    LICENSE\n    NOTICE\nclassifiers =\n    License :: OSI Approved :: MIT License\n[options]\ninstall_requires =\n    requests>=2\n",
			licenses:    []string{"MIT", "License :: OSI Approved :: MIT License"},
			licenseFile: "LICENSE",
		},
		{
			name:     "setup py",
			filename: "setup.py",
			content: `setup(
    name="example",
    version="1.0.0",
    license="MIT",
    classifiers=["License :: OSI Approved :: MIT License"],
    license_files=["LICENSE", "NOTICE"],
)`,
			licenses:    []string{"MIT", "License :: OSI Approved :: MIT License"},
			licenseFile: "LICENSE",
		},
		{
			name:     "gemspec array",
			filename: "example.gemspec",
			content:  `Gem::Specification.new { |s| s.name = "example"; s.version = "1.0.0"; s.licenses = ["MIT", "Apache-2.0"] }`,
			licenses: []string{"MIT", "Apache-2.0"},
		},
		{
			name:        "podspec hash",
			filename:    "example.podspec",
			content:     `Pod::Spec.new { |s| s.name = "Example"; s.version = "1.0"; s.license = { :type => "MIT", :file => "LICENSE" } }`,
			licenses:    []string{"MIT"},
			licenseFile: "LICENSE",
		},
		{
			name:     "composer array",
			filename: "composer.json",
			content:  `{"name":"example/package","version":"1.0.0","license":["MIT","Apache-2.0"]}`,
			licenses: []string{"MIT", "Apache-2.0"},
		},
		{
			name:     "maven",
			filename: "pom.xml",
			content: `<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>org.example</groupId>
  <artifactId>example</artifactId>
  <version>1.0.0</version>
  <licenses>
    <license><name>MIT</name><url>LICENSE.txt</url></license>
    <license><name>Apache-2.0</name><url>https://www.apache.org/licenses/LICENSE-2.0.txt</url></license>
  </licenses>
</project>`,
			licenses:    []string{"MIT", "Apache-2.0"},
			licenseFile: "LICENSE.txt",
		},
		{
			name:     "nuspec expression",
			filename: "example.nuspec",
			content:  `<package><metadata><id>Example</id><version>1.0.0</version><license type="expression">MIT OR Apache-2.0</license></metadata></package>`,
			licenses: []string{"MIT OR Apache-2.0"},
		},
		{
			name:        "nuspec file",
			filename:    "example.nuspec",
			content:     `<package><metadata><id>Example</id><version>1.0.0</version><license type="file">licenses/LICENSE.md</license></metadata></package>`,
			licenseFile: "licenses/LICENSE.md",
		},
		{
			name:        "cabal",
			filename:    "example.cabal",
			content:     "cabal-version: 3.0\nname: example\nversion: 1.0.0\nlicense: BSD-3-Clause\nlicense-files: LICENSE, NOTICE\n",
			licenses:    []string{"BSD-3-Clause"},
			licenseFile: "LICENSE",
		},
		{
			name:        "R DESCRIPTION",
			filename:    "DESCRIPTION",
			content:     "Package: example\nVersion: 1.0.0\nLicense: GPL-3 | BSD_3_clause + file LICENSE\n",
			licenses:    []string{"GPL-3", "BSD_3_clause"},
			licenseFile: "LICENSE",
		},
		{
			name:     "pub has no declaration",
			filename: "pubspec.yaml",
			content:  "name: example\nversion: 1.0.0\n",
		},
		{
			name:     "go mod has no declaration",
			filename: "go.mod",
			content:  "module example.com/project\n\ngo 1.25\n",
		},
		{
			name:     "mix package",
			filename: "mix.exs",
			content: `defmodule Example.MixProject do
  use Mix.Project
  def project do
    [app: :example, version: "1.0.0", package: package()]
  end
  defp package do
    [licenses: ["MIT", "Apache-2.0"]]
  end
end`,
			licenses: []string{"MIT", "Apache-2.0"},
		},
		{
			name:     "opam list",
			filename: "example.opam",
			content:  "opam-version: \"2.0\"\nname: \"example\"\nversion: \"1.0.0\"\nlicense: [\"MIT\" \"ISC\"]\ndepends: [\"ocaml\" {>= \"5.0\"} \"dune\"]\n",
			licenses: []string{"MIT", "ISC"},
		},
		{
			name:     "elm",
			filename: "elm.json",
			content:  `{"type":"package","name":"author/example","version":"1.0.0","license":"BSD-3-Clause","dependencies":{},"test-dependencies":{}}`,
			licenses: []string{"BSD-3-Clause"},
		},
		{
			name:     "bower array",
			filename: "bower.json",
			content:  `{"name":"example","version":"1.0.0","license":["MIT","Apache-2.0"]}`,
			licenses: []string{"MIT", "Apache-2.0"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Parse(tc.filename, []byte(tc.content))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !slices.Equal(result.Licenses, tc.licenses) {
				t.Errorf("Licenses = %q, want %q", result.Licenses, tc.licenses)
			}
			if result.LicenseFile != tc.licenseFile {
				t.Errorf("LicenseFile = %q, want %q", result.LicenseFile, tc.licenseFile)
			}
		})
	}
}

func TestParseName(t *testing.T) {
	// One row per manifest format that declares its own package
	// identity. Version is checked when the fixture has one; "" means
	// the format has a version field but this fixture omits it, "-"
	// means don't assert.
	testCases := []struct {
		path    string
		name    string
		version string
	}{
		{"testdata/golang/go.mod", "mod", ""},
		{"testdata/cargo/Cargo.toml", "update", "-"},
		{"testdata/npm/package.json", "librarian", "-"},
		{"testdata/npm/bower.json", "bootstrap", "-"},
		{"testdata/npm/deno.json", "my-deno-app", "-"},
		{"testdata/composer/composer.json", "laravel/laravel", "-"},
		{"testdata/pub/pubspec.yaml", "angular", "-"},
		{"testdata/crystal/shard.yml", "registry", "-"},
		{"testdata/gleam/gleam.toml", "my_gleam_app", "-"},
		{"testdata/julia/Project.toml", "MyProject", "-"},
		{"testdata/dub/dub.json", "ddox", "-"},
		{"testdata/haxelib/haxelib.json", "flixel", "-"},
		{"testdata/vcpkg/vcpkg.json", "warzone2100", "-"},
		{"testdata/cpan/META.json", "App-rainbarf", "-"},
		{"testdata/cran/DESCRIPTION", "ggplot2", "-"},
		{"testdata/hackage/example.cabal", "cabal-parser", "-"},
		// elm-package.json fixture is an application (no name field)
		{"testdata/elm/elm-package.json", "", "-"},
		{"testdata/gem/devise.gemspec", "devise", ""},
		{"testdata/cocoapods/example.podspec", "CocoaLumberjack", "-"},
		{"testdata/hex/mix.exs", "mixup", "-"},
		{"testdata/pypi/pyproject.toml", "tidelift", "-"},
		{"testdata/pypi/setup.py", "political-memory", "-"},
		{"testdata/swift/Package.swift", "swift-package-converter", ""},
		{"testdata/conan/conanfile.py", "mypackage", "-"},
		{"testdata/clojure/project.clj", "clojars-json", "-"},
		{"testdata/maven/pom.xml", "-", "-"},
		{"testdata/maven/ivy.xml", "-", "-"},
		{"testdata/maven/build.sbt", "scala-parser-combinators", "-"},
		{"testdata/nuget/example.nuspec", "Bottles", "-"},
		{"testdata/nuget/Example.csproj", "Example", "-"},
		{"testdata/bazel/MODULE.bazel", "elemental2", "-"},
		{"testdata/lake/lakefile.lean", "example", ""},
		{"testdata/luarocks/example.rockspec", "-", "-"},
		{"testdata/nimble/example.nimble", "example", "-"},
		{"testdata/alpine/APKBUILD", "curl", "-"},
		{"testdata/arch/PKGBUILD", "-", "-"},
		{"testdata/rpm/hello.spec", "hello", "2.10"},

		// Formats with no self-name should stay empty.
		{"testdata/gem/Gemfile", "", ""},
		{"testdata/pypi/requirements.txt", "", ""},
		{"testdata/golang/go.sum", "", ""},
		{"testdata/cargo/Cargo.lock", "", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			content, err := os.ReadFile(tc.path)
			if err != nil {
				t.Skipf("fixture not found: %v", err)
			}
			result, err := Parse(filepath.Base(tc.path), content)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if tc.name == "-" {
				if result.Name == "" {
					t.Errorf("Name is empty; expected some value")
				}
			} else if result.Name != tc.name {
				t.Errorf("Name = %q, want %q", result.Name, tc.name)
			}
			if tc.version != "-" && result.Version != tc.version {
				t.Errorf("Version = %q, want %q", result.Version, tc.version)
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
		{"go.sum", "golang", Supplement, true},

		// pypi
		{"requirements.txt", "pypi", Manifest, true},
		{"requirements-dev.txt", "pypi", Manifest, true},
		{"requirements/test.txt", "pypi", Manifest, true},
		{"requirements.frozen", "pypi", Manifest, true},
		{"requirements.rb", "", "", false},
		{"Pipfile", "pypi", Manifest, true},
		{"Pipfile.lock", "pypi", Lockfile, true},
		{"pyproject.toml", "pypi", Manifest, true},
		{"poetry.lock", "pypi", Lockfile, true},

		// maven
		{"pom.xml", "maven", Manifest, true},

		// composer
		{"composer.json", "composer", Manifest, true},
		{"composer.lock", "composer", Lockfile, true},

		// lean
		{"lakefile.toml", "lean", Manifest, true},
		{"lakefile.lean", "lean", Manifest, true},
		{"lake-manifest.json", "lean", Lockfile, true},

		// docker
		{"Dockerfile", "docker", Manifest, true},
		{"docker-compose.yml", "docker", Manifest, true},

		// github-actions
		{".github/workflows/ci.yml", "github-actions", Manifest, true},
		{".github/workflows/actions.lock", "github-actions", Lockfile, true},

		// helm
		{"Chart.yaml", "helm", Manifest, true},
		{"Chart.lock", "helm", Lockfile, true},

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

func TestSwiftSourcePURLs(t *testing.T) {
	content, err := os.ReadFile("testdata/swift/Package.resolved.2")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	result, err := Parse("Package.resolved", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := map[string]string{
		"github.com/krzyzanowskim/CryptoSwift": "pkg:swift/github.com/krzyzanowskim/CryptoSwift@1.6.0",
		"github.com/apple/swift-docc-plugin":   "pkg:swift/github.com/apple/swift-docc-plugin@1.0.0",
		"apple.swift-argument-parser":          "",
	}
	if len(result.Dependencies) != len(want) {
		t.Fatalf("Dependencies has %d entries, want %d: %+v", len(result.Dependencies), len(want), result.Dependencies)
	}
	for _, dependency := range result.Dependencies {
		wantPURL, ok := want[dependency.Name]
		if !ok {
			t.Errorf("unexpected dependency: %+v", dependency)
			continue
		}
		if dependency.PURL != wantPURL {
			t.Errorf("%s PURL = %q, want %q", dependency.Name, dependency.PURL, wantPURL)
		}
	}
}

func TestHelmPURLsAndDigest(t *testing.T) {
	chartContent, err := os.ReadFile("testdata/helm/Chart.yaml")
	if err != nil {
		t.Fatalf("read chart fixture: %v", err)
	}
	chart, err := Parse("Chart.yaml", chartContent)
	if err != nil {
		t.Fatalf("parse Chart.yaml: %v", err)
	}
	if chart.Name != "example-chart" || chart.Version != "1.2.3" {
		t.Errorf("chart identity = %q %q, want example-chart 1.2.3", chart.Name, chart.Version)
	}

	chartDependencies := make(map[string]Dependency, len(chart.Dependencies))
	for _, dependency := range chart.Dependencies {
		chartDependencies[dependency.Name] = dependency
	}
	wantChartPURLs := map[string]string{
		"postgresql":     "pkg:helm/postgresql",
		"redis":          "pkg:helm/redis",
		"metrics-server": "pkg:helm/metrics-server",
		"common":         "pkg:helm/common",
		"local-chart":    "pkg:helm/local-chart",
		"plugin-chart":   "pkg:helm/plugin-chart",
	}
	for name, want := range wantChartPURLs {
		dependency, ok := chartDependencies[name]
		if !ok {
			t.Errorf("Chart.yaml missing dependency %q", name)
			continue
		}
		if dependency.PURL != want {
			t.Errorf("Chart.yaml %s PURL = %q, want %q", name, dependency.PURL, want)
		}
	}
	postgresql := chartDependencies["postgresql"]
	if postgresql.RegistryURL != "https://charts.bitnami.com/bitnami" {
		t.Errorf("Chart.yaml postgresql RegistryURL = %q", postgresql.RegistryURL)
	}
	localChart := chartDependencies["local-chart"]
	wantSource := Source{Kind: SourcePath, Value: "../local-chart"}
	if localChart.RegistryURL != "" || localChart.Source != wantSource {
		t.Errorf("Chart.yaml local-chart RegistryURL/Source = %q %+v, want empty %+v",
			localChart.RegistryURL, localChart.Source, wantSource)
	}

	lockContent, err := os.ReadFile("testdata/helm/Chart.lock")
	if err != nil {
		t.Fatalf("read lock fixture: %v", err)
	}
	lock, err := Parse("Chart.lock", lockContent)
	if err != nil {
		t.Fatalf("parse Chart.lock: %v", err)
	}
	if lock.Digest != "sha256:8ca45f73ae3f6170a09b64a967006e98e13cd91eb51e5ab0599bb87296c7df0a" {
		t.Errorf("Chart.lock Digest = %q", lock.Digest)
	}

	lockDependencies := make(map[string]Dependency, len(lock.Dependencies))
	for _, dependency := range lock.Dependencies {
		lockDependencies[dependency.Name] = dependency
	}
	wantLockPURLs := map[string]string{
		"postgresql":     "pkg:helm/postgresql@12.1.15",
		"redis":          "pkg:helm/redis@17.3.7",
		"metrics-server": "pkg:helm/metrics-server@3.12.2",
		"common":         "pkg:helm/common@1.17.1",
		"local-chart":    "pkg:helm/local-chart@0.1.0",
		"plugin-chart":   "pkg:helm/plugin-chart@2.0.0",
	}
	for name, want := range wantLockPURLs {
		dependency, ok := lockDependencies[name]
		if !ok {
			t.Errorf("Chart.lock missing dependency %q", name)
			continue
		}
		if dependency.PURL != want {
			t.Errorf("Chart.lock %s PURL = %q, want %q", name, dependency.PURL, want)
		}
	}
	postgresql = lockDependencies["postgresql"]
	if postgresql.Integrity != "" {
		t.Errorf("Chart.lock postgresql Integrity = %q, want empty", postgresql.Integrity)
	}
}

func TestParsePEP508ParenthesizedRequirements(t *testing.T) {
	content, err := os.ReadFile("testdata/pypi/pep508-parenthesized/pyproject.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	result, err := Parse("pyproject.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if len(result.Dependencies) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(result.Dependencies))
	}

	expected := map[string]struct {
		version string
		purl    string
	}{
		"requests":     {version: ">=2.26,<3.0", purl: "pkg:pypi/requests"},
		"packaging":    {version: ">=24.2", purl: "pkg:pypi/packaging"},
		"platformdirs": {version: ">=3.0.0,<5", purl: "pkg:pypi/platformdirs"},
		"importlib-metadata": {
			version: ">=6.0",
			purl:    "pkg:pypi/importlib-metadata",
		},
	}

	dependencies := make(map[string]Dependency, len(result.Dependencies))
	for _, dependency := range result.Dependencies {
		if _, exists := dependencies[dependency.Name]; exists {
			t.Errorf("duplicate dependency %q", dependency.Name)
		}
		dependencies[dependency.Name] = dependency
	}

	for name, want := range expected {
		dependency, ok := dependencies[name]
		if !ok {
			t.Errorf("expected dependency %q", name)
			continue
		}
		if dependency.Version != want.version {
			t.Errorf("%s version = %q, want %q", dependency.Name, dependency.Version, want.version)
		}
		if dependency.PURL != want.purl {
			t.Errorf("%s PURL = %q, want %q", dependency.Name, dependency.PURL, want.purl)
		}
	}
}

func TestRegistryURLNotIncludedForDefaultRegistry(t *testing.T) {
	// Test that default registry URLs don't add repository_url qualifier
	testCases := []struct {
		name       string
		content    string
		filename   string
		wantInPURL bool
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
