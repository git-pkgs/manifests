package git

import (
	"os"
	"reflect"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGitmodules(t *testing.T) {
	content, err := os.ReadFile("../../testdata/git/.gitmodules")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gitmodulesParser{}
	res, err := parser.Parse(".gitmodules", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	want := []core.Dependency{
		{Name: "github.com/example/foo", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/foo.git"}},
		{Name: "github.com/example/bar", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "git@github.com:example/bar.git"}},
		{Name: "gitlab.com/example/baz", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "https://gitlab.com/example/baz.git", Branch: "main"}},
		{Name: "deps/sibling", Scope: core.Runtime, Direct: true, NoPURL: true,
			Source: core.Source{Kind: core.SourceGit, Value: "../sibling.git"}},
	}
	if len(res.Dependencies) != len(want) {
		t.Fatalf("got %d dependencies, want %d: %+v", len(res.Dependencies), len(want), res.Dependencies)
	}
	for i, dep := range res.Dependencies {
		if !reflect.DeepEqual(dep, want[i]) {
			t.Errorf("dependency %d = %+v, want %+v", i, dep, want[i])
		}
	}
}

func TestGitmodulesWithoutURL(t *testing.T) {
	parser := &gitmodulesParser{}
	res, err := parser.Parse(".gitmodules", []byte("[submodule \"foo\"]\n\tpath = foo\n"))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{{Name: "foo", Scope: core.Runtime, Direct: true, NoPURL: true}}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}

func TestGitmodulesQuotedValues(t *testing.T) {
	content := "[submodule \"lib/foo\"]\n" +
		"\tpath = \"lib/foo\"\n" +
		"\turl = \"https://github.com/example/foo.git\" ; mirror\n" +
		"\tbranch = \"fix#1\" # pinned\n" +
		"[submodule \"bar\"]\r\n" +
		"\tpath = bar\r\n" +
		"\turl = https://github.com/example/\\\r\n" +
		"bar.git\r\n"
	res, err := (&gitmodulesParser{}).Parse(".gitmodules", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{
		{Name: "github.com/example/foo", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/foo.git", Branch: "fix#1"}},
		{Name: "github.com/example/bar", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/bar.git"}},
	}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}

func TestConfigValue(t *testing.T) {
	tests := []struct {
		in, want string
		more, ok bool
	}{
		{`plain`, "plain", false, true},
		{`"quoted value"`, "quoted value", false, true},
		{`a "b c" d`, "a b c d", false, true},
		{`trailing   ; comment`, "trailing", false, true},
		{`"kept  "  # comment`, "kept  ", false, true},
		{`esc\"aped\\ \t\n`, "esc\"aped\\ \t\n", false, true},
		{`"open`, "", false, false},
		{`bad\q`, "", false, false},
		{`ends with \`, "", true, true},
		{`# only a comment \`, "", false, true},
	}
	for _, tt := range tests {
		got, more, ok := configValue(tt.in)
		if got != tt.want || more != tt.more || ok != tt.ok {
			t.Errorf("configValue(%q) = %q, %v, %v; want %q, %v, %v", tt.in, got, more, ok, tt.want, tt.more, tt.ok)
		}
	}
}
