package manifests

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestGoModReplacements(t *testing.T) {
	content := []byte(`module example.com/app

go 1.26

require (
	example.com/library v1.0.0
	example.com/forked v1.2.0
	example.com/local v0.1.0
	"example.com/quoted" v2.0.0//note
	example.com/kept v3.0.0 // indirect
)

replace example.com/library => example.com/library v1.1.0

replace (
	example.com/forked v1.2.0 => github.com/me/forked v1.2.1 // fork
	example.com/forked => github.com/me/forked v1.3.0
	example.com/local => ../local
	example.com/local v0.1.0 => "C:\\src\\local"
	example.com/quoted => "../dir with spaces"
	example.com/versionless => example.com/other
	example.com/missing =>
)
`)
	result, err := Parse("go.mod", content)
	if err != nil {
		t.Fatal(err)
	}

	wantDeps := map[string]string{
		"example.com/library": "v1.0.0",
		"example.com/forked":  "v1.2.0",
		"example.com/local":   "v0.1.0",
		"example.com/quoted":  "v2.0.0",
		"example.com/kept":    "v3.0.0",
	}
	if len(result.Dependencies) != len(wantDeps) {
		t.Fatalf("dependencies = %+v", result.Dependencies)
	}
	for _, dep := range result.Dependencies {
		if wantDeps[dep.Name] != dep.Version {
			t.Errorf("dependency %s %s, want version %q", dep.Name, dep.Version, wantDeps[dep.Name])
		}
	}

	path := func(dir string) Source { return Source{Kind: SourcePath, Value: dir} }
	want := []Replacement{
		{Name: "example.com/library", ReplacementName: "example.com/library", ReplacementVersion: "v1.1.0", Location: "replace/example.com%2Flibrary"},
		{Name: "example.com/forked", Version: "v1.2.0", ReplacementName: "github.com/me/forked", ReplacementVersion: "v1.2.1", Location: "replace/example.com%2Fforked"},
		{Name: "example.com/forked", ReplacementName: "github.com/me/forked", ReplacementVersion: "v1.3.0", Location: "replace/example.com%2Fforked/2"},
		{Name: "example.com/local", Source: path("../local"), Location: "replace/example.com%2Flocal"},
		{Name: "example.com/local", Version: "v0.1.0", Source: path(`C:\src\local`), Location: "replace/example.com%2Flocal/2"},
		{Name: "example.com/quoted", Source: path("../dir with spaces"), Location: "replace/example.com%2Fquoted"},
		{Name: "example.com/versionless", ReplacementName: "example.com/other", Location: "replace/example.com%2Fversionless"},
	}
	if !reflect.DeepEqual(result.Replacements, want) {
		t.Errorf("replacements =\n%+v\nwant\n%+v", result.Replacements, want)
	}

	// Declarations still omit replaced modules, so the quoted require must
	// decode to match the unquoted replace.
	if len(result.Declarations) != 1 || result.Declarations[0].Name != "example.com/kept" {
		t.Errorf("declarations = %+v", result.Declarations)
	}
}

func TestGoModWithoutReplacements(t *testing.T) {
	result, err := Parse("go.mod", []byte("module example.com/app\n\nrequire example.com/a v1.0.0\n"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Replacements != nil {
		t.Errorf("replacements = %+v, want none", result.Replacements)
	}
}

func TestGoModParenthesisTokens(t *testing.T) {
	content := []byte(`module example.com/app

require(
	example.com/a v1.0.0
)

require ( )

require()

replace(
	example.com/a => ../a
)

replace ( )

replace()
`)
	result, err := Parse("go.mod", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "example.com/a" || result.Dependencies[0].Version != "v1.0.0" {
		t.Errorf("dependencies = %+v", result.Dependencies)
	}
	want := []Replacement{{Name: "example.com/a", Source: Source{Kind: SourcePath, Value: "../a"}, Location: "replace/example.com%2Fa"}}
	if !reflect.DeepEqual(result.Replacements, want) {
		t.Errorf("replacements = %+v, want %+v", result.Replacements, want)
	}
	if len(result.Declarations) != 0 {
		t.Errorf("declarations = %+v, want none", result.Declarations)
	}
}

func TestGoModEmptyOneLineBlockKeepsLaterDirectives(t *testing.T) {
	content := []byte("module example.com/app\n\nrequire ( )\nrequire example.com/b v2.0.0\n\nreplace ( )\nreplace example.com/b => ../b\n")
	result, err := Parse("go.mod", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "example.com/b" {
		t.Errorf("dependencies = %+v", result.Dependencies)
	}
	if len(result.Replacements) != 1 || result.Replacements[0].Name != "example.com/b" {
		t.Errorf("replacements = %+v", result.Replacements)
	}
}

func TestGoModLongLine(t *testing.T) {
	const tokens = 1 << 20
	content := []byte("module example.com/app\n\nrequire example.com/a v1.0.0" + strings.Repeat(" a", tokens) + "\n")
	started := time.Now()
	result, err := Parse("go.mod", content)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Errorf("parsing a %d-token line took %v", tokens, elapsed)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Version != "v1.0.0" {
		t.Errorf("dependencies = %+v", result.Dependencies)
	}
}
