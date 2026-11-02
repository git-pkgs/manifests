package opam

import (
	"reflect"
	"slices"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestParse(t *testing.T) {
	content := []byte(`opam-version: "2.0"
name: "example"
version: "1.2.3"
license: ["MIT" "ISC"]
depends: [
  "ocaml" {>= "5.0"}
  # "commented-out"
  "dune" {build}
  "alcotest" {with-test & >= "1.8.0"}
  "cmdliner" {>= "1.0" & < "2.0"}
  "unix" {os != "win32"}
  "ocamlformat" {with-dev-setup} {>= "0.27.0"}
]
`)

	result, err := (&parser{}).Parse("example.opam", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Name != "example" {
		t.Errorf("Name = %q, want example", result.Name)
	}
	if result.Version != "1.2.3" {
		t.Errorf("Version = %q, want 1.2.3", result.Version)
	}
	if !slices.Equal(result.Licenses, []string{"MIT", "ISC"}) {
		t.Errorf("Licenses = %q, want [MIT ISC]", result.Licenses)
	}
	if len(result.Dependencies) != 6 {
		t.Fatalf("Dependencies = %d, want 6", len(result.Dependencies))
	}
	want := []core.Dependency{
		{Name: "ocaml", Version: ">= 5.0", Scope: core.Runtime, Direct: true},
		{Name: "dune", Scope: core.Build, Direct: true},
		{Name: "alcotest", Version: ">= 1.8.0", Scope: core.Test, Direct: true},
		{Name: "cmdliner", Version: ">= 1.0 & < 2.0", Scope: core.Runtime, Direct: true},
		{Name: "unix", Scope: core.Runtime, Direct: true},
		{Name: "ocamlformat", Version: ">= 0.27.0", Scope: core.Development, Direct: true},
	}
	if !reflect.DeepEqual(result.Dependencies, want) {
		t.Errorf("Dependencies = %#v, want %#v", result.Dependencies, want)
	}
}

func TestScalarLicense(t *testing.T) {
	result, err := (&parser{}).Parse("opam", []byte(`license: "Apache-2.0"`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !slices.Equal(result.Licenses, []string{"Apache-2.0"}) {
		t.Errorf("Licenses = %q, want [Apache-2.0]", result.Licenses)
	}
}
