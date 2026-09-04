package vagrant

import (
	"fmt"
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestVagrantfileParser(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile("../../testdata/vagrant/Vagrantfile")
	if err != nil {
		t.Fatal(err)
	}
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{
		name:      "hashicorp/bionic64",
		version:   ">= 1.0, < 2.0",
		integrity: "sha256-7c222fb2927d828af22f592134e8932480637c0d1a013007ed9b92a65e53724e",
		source: core.Source{
			Kind:  core.SourceURL,
			Value: "https://boxes.example.test/bionic64.json#catalog",
		},
	})
}

func TestVagrantfileParserExactVersionAndMissingOptionalFields(t *testing.T) {
	t.Parallel()

	content := []byte(`
Vagrant.configure("2") do |config|
  config.vm.box = 'ubuntu/jammy64'
  config.vm.box_version = '20241002.0.0'
end
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{
		name:    "ubuntu/jammy64",
		version: "20241002.0.0",
	})
}

func TestVagrantfileParserSupportsFileURL(t *testing.T) {
	t.Parallel()

	content := []byte(`
config.vm.box = "local/example"
config.vm.box_url = "file:///srv/boxes/example.box"
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{
		name: "local/example",
		source: core.Source{
			Kind:  core.SourceURL,
			Value: "file:///srv/boxes/example.box",
		},
	})
}

func TestVagrantfileParserChecksumTypes(t *testing.T) {
	t.Parallel()

	for _, checksumType := range []string{"md5", "sha1", "sha256", "sha384", "sha512"} {
		t.Run(checksumType, func(t *testing.T) {
			t.Parallel()
			content := []byte(fmt.Sprintf(`
config.vm.box = "example/box"
config.vm.box_download_checksum = "abc123"
config.vm.box_download_checksum_type = %q
`, checksumType))
			result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
			if err != nil {
				t.Fatal(err)
			}
			assertVagrantBox(t, result, vagrantBoxExpectation{
				name:      "example/box",
				integrity: checksumType + "-abc123",
			})
		})
	}
}

func TestVagrantfileParserRequiresBothChecksumFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		assignments string
	}{
		{name: "missing type", assignments: `config.vm.box_download_checksum = "abc123"`},
		{name: "missing checksum", assignments: `config.vm.box_download_checksum_type = "sha256"`},
		{name: "dynamic type", assignments: `
config.vm.box_download_checksum = "abc123"
config.vm.box_download_checksum_type = ENV.fetch("CHECKSUM_TYPE")`},
		{name: "dynamic checksum", assignments: `
config.vm.box_download_checksum = checksum_for_box()
config.vm.box_download_checksum_type = "sha256"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			content := []byte("config.vm.box = \"example/box\"\n" + test.assignments + "\n")
			result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
			if err != nil {
				t.Fatal(err)
			}
			assertVagrantBox(t, result, vagrantBoxExpectation{name: "example/box"})
		})
	}
}

func TestVagrantfileParserSkipsDynamicAssignments(t *testing.T) {
	t.Parallel()

	content := []byte(`
config.vm.box = box_name
config.vm.box = "#{organization}/dynamic"
config.vm.box = "#$global_box"
config.vm.box = "#@instance_box"
config.vm.box = ENV.fetch("VAGRANT_BOX")
config.vm.box = "joined/box" + suffix
config.vm.box = ["first/box", "second/box"]
config.vm.box_version = version_constraint()
config.vm.box_url = ["https://one.example", "https://two.example"]
config.vm.box_download_checksum = checksum
config.vm.box_download_checksum_type = ENV["CHECKSUM_TYPE"]
config.vm.box_check_update = false
other.config.vm.box = "wrong/prefix"

config . vm . box = "literal/box" # retained after dynamic assignments
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{name: "literal/box"})
}

func TestVagrantfileParserSkipsInvalidTrailingExpressions(t *testing.T) {
	t.Parallel()

	content := []byte(`
config.vm.box = "conditional/box" if enabled
config.vm.box = "frozen/box".freeze
config.vm.box = "unterminated/box
config.vm.box = "literal/box" # valid
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{name: "literal/box"})
}

func TestVagrantfileParserIgnoresNonCodeRegions(t *testing.T) {
	t.Parallel()

	content := []byte(`
config.vm.provision "shell", inline: <<~SHELL
config.vm.box = "heredoc/box"
SHELL

=begin documentation
config.vm.box = "commented/box"
=end

config.vm.box = "literal/box"
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{name: "literal/box"})
}

func TestVagrantfileParserWithoutLiteralBox(t *testing.T) {
	t.Parallel()

	content := []byte(`
config.vm.box = ENV.fetch("VAGRANT_BOX")
config.vm.box_version = "~> 1.0"
config.vm.box_url = "https://boxes.example.test/catalog.json"
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 0 || len(result.Declarations) != 0 {
		t.Errorf("result = %+v, want no box declarations", result)
	}
}

func TestRubyStringEscapes(t *testing.T) {
	t.Parallel()

	content := []byte(`
config.vm.box = 'literal\q/box'
config.vm.box_version = "line\nconstraint"
config.vm.box_url = "https://boxes.example.test/escaped\#fragment"
`)
	result, err := (&vagrantfileParser{}).Parse("Vagrantfile", content)
	if err != nil {
		t.Fatal(err)
	}
	assertVagrantBox(t, result, vagrantBoxExpectation{
		name:    "literal\\q/box",
		version: "line\nconstraint",
		source: core.Source{
			Kind:  core.SourceURL,
			Value: "https://boxes.example.test/escaped#fragment",
		},
	})
}

type vagrantBoxExpectation struct {
	name      string
	version   string
	integrity string
	source    core.Source
}

func assertVagrantBox(t *testing.T, result *core.Result, want vagrantBoxExpectation) {
	t.Helper()
	if len(result.Dependencies) != 1 {
		t.Fatalf("dependencies = %+v, want one", result.Dependencies)
	}
	dependency := result.Dependencies[0]
	if dependency.Name != want.name || dependency.Version != want.version ||
		dependency.Integrity != want.integrity || dependency.Source != want.source ||
		dependency.Scope != core.Runtime || !dependency.Direct || dependency.RegistryURL != "" {
		t.Errorf("dependency = %+v, want %+v", dependency, want)
	}
	if len(result.Declarations) != 1 {
		t.Fatalf("declarations = %+v, want one", result.Declarations)
	}
	declaration := result.Declarations[0]
	if declaration.Name != want.name || declaration.Version != want.version ||
		declaration.Source != want.source || declaration.Scope != core.Runtime ||
		!declaration.Direct || declaration.Location != boxLocation {
		t.Errorf("declaration = %+v, want %+v at %q", declaration, want, boxLocation)
	}
}
