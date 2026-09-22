package manifests_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/git-pkgs/manifests"
)

func TestParseScripts(t *testing.T) {
	tests := []struct {
		file       string
		dependency string
		scripts    map[string][]string
	}{
		{"npm/package.json", "bindings", map[string][]string{
			"preinstall": {"node scripts/check.js"}, "install": {"node-gyp rebuild"},
			"postinstall": {"node scripts/setup.js"}, "test": {"node --test"},
		}},
		{"composer/composer.json", "example/library", map[string][]string{
			"post-install-cmd": {`Example\Installer::install`, "@php scripts/setup.php"},
			"post-update-cmd":  {"@post-install-cmd"}, "test": {"php tests/run.php"},
		}},
		{"cargo/Cargo.toml", "libc", map[string][]string{"build": {"scripts/build.rs"}}},
		{"gem/native.gemspec", "rake", map[string][]string{"extensions": {
			"ext/widget/extconf.rb", "ext/helper/extconf.rb", "ext/extra/extconf.rb", "ext/final/extconf.rb",
		}}},
		{"crystal/shard.yml", "openssl", map[string][]string{"postinstall": {"make native\nmake copy\n"}}},
		{"deno/deno.json", "chalk", map[string][]string{"build": {"deno run -A build.ts"}, "test": {"deno test"}}},
		{"dub/dub.json", "vibe-d", map[string][]string{
			"preBuildCommands":                         {"echo preparing", "make native"},
			"postBuildCommands-windows":                {"copy widget.dll bin"},
			"configurations/release/postBuildCommands": {"strip widget"},
			"buildTypes/coverage/preRunCommands":       {"mkdir -p coverage"},
			"subPackages/helper/preGenerateCommands":   {"generate helper"},
		}},
		{"nuget/project.json", "Example.Library", map[string][]string{
			"precompile": {"echo preparing", "generate-code"}, "postcompile": {"copy-assets"},
		}},
		{"nuget/example.csproj", "Example.Library", map[string][]string{
			"PreBuildEvent": {"echo preparing"}, "PostBuildEvent": {"echo complete"},
			"Target/Generate": {`echo "generating"`, "generate-code"},
		}},
		{"pypi/pyproject.toml", "cython", map[string][]string{"tool.poetry.build": {"scripts/build.py"}}},
		{"opam/widget.opam", "dune", map[string][]string{
			"build":   {"[\n  [\"dune\" \"build\" \"-p\" name \"-j\" jobs]\n  [\"dune\" \"runtest\"] {with-test}\n]"},
			"install": {`[[make "install"]]`},
		}},
		{"alpine/APKBUILD", "musl", map[string][]string{"install": {"$pkgname.pre-install", "$pkgname.post-install"}}},
		{"arch/PKGBUILD", "glibc", map[string][]string{"install": {"widget.install"}}},
		{"cocoapods/Widget.podspec", "Helper", map[string][]string{"prepare_command": {"    make native\n    cp native.a lib/\n"}}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join("testdata/scripts", tt.file))
			if err != nil {
				t.Fatal(err)
			}
			result, err := manifests.Parse(filepath.Base(tt.file), content)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Scripts, tt.scripts) {
				t.Fatalf("Scripts = %#v, want %#v", result.Scripts, tt.scripts)
			}
			if len(result.Dependencies) != 1 || result.Dependencies[0].Name != tt.dependency {
				t.Fatalf("Dependencies = %#v, want %s", result.Dependencies, tt.dependency)
			}
		})
	}
}

func TestParseScriptVariants(t *testing.T) {
	tests := []struct {
		name, file, content string
		scripts             map[string][]string
	}{
		{"cargo enabled", "Cargo.toml", "[package]\nbuild = true", map[string][]string{"build": {"build.rs"}}},
		{"cargo disabled", "Cargo.toml", "[package]\nbuild = false", nil},
		{"cargo implicit", "Cargo.toml", "[package]\nname = 'example'", nil},
		{"poetry shorthand", "pyproject.toml", "[tool.poetry]\nbuild = 'build.py'", map[string][]string{"tool.poetry.build": {"build.py"}}},
		{"pdm custom hook", "pyproject.toml", "[tool.pdm.build]\ncustom-hook = 'build.py'", map[string][]string{"tool.pdm.build.custom-hook": {"build.py"}}},
		{"pdm implicit hook", "pyproject.toml", "[build-system]\nbuild-backend = 'pdm.backend'", nil},
		{"gem words", "example.gemspec", "s.extensions = %w(ext/a.rb ext/b.rb)", map[string][]string{"extensions": {"ext/a.rb", "ext/b.rb"}}},
		{"gem dynamic", "example.gemspec", "s.extensions = Dir['ext/**/extconf.rb']", nil},
		{"gem crlf", "example.gemspec", "s.extensions = ['ext/a.rb']\r\n", map[string][]string{"extensions": {"ext/a.rb"}}},
		{"gem comment", "example.gemspec", "# s.extensions = ['ext/a.rb']", nil},
		{"podspec literal", "example.podspec", "s.prepare_command = 'make native'", map[string][]string{"prepare_command": {"make native"}}},
		{"podspec comment", "example.podspec", "# s.prepare_command = 'make native'", nil},
		{"podspec dynamic", "example.podspec", "s.prepare_command = File.read('prepare.sh')", nil},
		{"podspec quoted earlier occurrence", "example.podspec", "x = \"s.prepare_command = 'make native'\"\ns.prepare_command = 'make native'", map[string][]string{"prepare_command": {"make native"}}},
		{"podspec declaration order", "example.podspec", "s.prepare_command = <<-CMD\nmake first\nCMD\ns.prepare_command = 'make second'", map[string][]string{"prepare_command": {"make first\n", "make second"}}},
		{"podspec unterminated heredoc", "example.podspec", "s.prepare_command = <<-CMD\nmake first", nil},
		{"arch unquoted", "PKGBUILD", "install=$pkgname.install", map[string][]string{"install": {"$pkgname.install"}}},
		{"arch quoted", "PKGBUILD", "install=\"$pkgname.install\"", map[string][]string{"install": {"$pkgname.install"}}},
		{"alpine single quoted", "APKBUILD", "install='a.pre-install a.post-install'", map[string][]string{"install": {"a.pre-install", "a.post-install"}}},
		{"shell comment", "APKBUILD", "# install='a.pre-install'", nil},
		{"shell partial expression", "PKGBUILD", "install='a'\".install\"", nil},
		{"composer empty", "composer.json", `{"scripts":{"post-install-cmd":[],"test":null}}`, nil},
		{"npm metadata", "package.json", `{"scripts":{"test":"node --test","description":false}}`, map[string][]string{"test": {"node --test"}}},
		{"deno task graph", "deno.json", `{"tasks":{"all":{"dependencies":["build"]}}}`, nil},
		{"opam filters", "example.opam", "build: [[\"echo\" \"]\"] {os = \"linux\"}]\nremove: [ ]", map[string][]string{"build": {`[["echo" "]"] {os = "linux"}]`}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manifests.Parse(tt.file, []byte(tt.content))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(result.Scripts, tt.scripts) {
				t.Fatalf("Scripts = %#v, want %#v", result.Scripts, tt.scripts)
			}
		})
	}
}

func TestParseWithoutScripts(t *testing.T) {
	for _, file := range []string{
		"package.json", "composer.json", "deno.json", "dub.json", "project.json", "package-lock.json", "composer.lock",
		"shard.yml", "Cargo.toml", "pyproject.toml", "example.gemspec", "example.podspec", "APKBUILD", "PKGBUILD", "example.opam",
	} {
		t.Run(file, func(t *testing.T) {
			content := ""
			if filepath.Ext(file) == ".json" || file == "composer.lock" {
				content = "{}"
			}
			result, err := manifests.Parse(file, []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if result.Scripts != nil {
				t.Fatalf("Scripts = %#v, want nil", result.Scripts)
			}
		})
	}
}
