package git

import (
	"os"
	"reflect"
	"strings"
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
		n        int
		ok       bool
	}{
		{`plain`, "plain", 5, true},
		{"plain\nnext", "plain", 6, true},
		{`"quoted value"`, "quoted value", 14, true},
		{`a "b c" d`, "a b c d", 9, true},
		{`trailing   ; comment`, "trailing", 20, true},
		{`"kept  "  # comment`, "kept  ", 19, true},
		{`esc\"aped\\ \t\n`, "esc\"aped\\ \t\n", 16, true},
		{"a\\\nb\nc", "ab", 5, true},
		{"x\\\r\ny\r\nnext", "xy", 7, true},
		{`ends with \`, "ends with ", 11, true},
		{"# comment \\\nnext", "", 12, true},
		{"a \v", "a \v", 3, true},
		{"a \f\r", "a \f", 4, true},
		{`"open`, "", 5, false},
		{"\"open\nnext", "", 6, false},
		{"bad\\q rest\nnext", "", 11, false},
	}
	for _, tt := range tests {
		got, n, ok := configValue(tt.in)
		if got != tt.want || n != tt.n || ok != tt.ok {
			t.Errorf("configValue(%q) = %q, %d, %v; want %q, %d, %v", tt.in, got, n, ok, tt.want, tt.n, tt.ok)
		}
	}
}

func TestGitmodulesManyContinuationLines(t *testing.T) {
	const lines = 5000
	content := "[submodule \"one\"]\n\tpath = one\n\turl = https://github.com/example/one.git\n\tbranch = " +
		strings.Repeat("a\\\n", lines) + "\n[submodule \"two\"]\n\tpath = two\n"
	res, err := (&gitmodulesParser{}).Parse(".gitmodules", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{{Name: "github.com/example/one", Scope: core.Runtime, Direct: true,
		Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/one.git", Branch: strings.Repeat("a", lines)}},
		{Name: "two", Scope: core.Runtime, Direct: true, NoPURL: true}}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %d deps, want the branch joined from %d continuation lines and the submodule after it", len(res.Dependencies), lines)
	}
}

func TestGitmodulesKeysIgnoreCase(t *testing.T) {
	content := "[submodule \"one\"]\n" +
		"\tPath = one\n" +
		"\tURL = https://github.com/example/one.git\n" +
		"\tBranch = Main\n"
	res, err := (&gitmodulesParser{}).Parse(".gitmodules", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{{Name: "github.com/example/one", Scope: core.Runtime, Direct: true,
		Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/one.git", Branch: "Main"}}}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}

func TestGitmodulesCommentLineEndingInBackslash(t *testing.T) {
	content := "[submodule \"one\"]\n" +
		"\tpath = one\n" +
		"# example = \\\n" +
		"[submodule \"two\"]\n" +
		"\tpath = two\n" +
		"\t; note = \\\n" +
		"\turl = https://github.com/example/two.git\n"
	res, err := (&gitmodulesParser{}).Parse(".gitmodules", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{
		{Name: "one", Scope: core.Runtime, Direct: true, NoPURL: true},
		{Name: "github.com/example/two", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/two.git"}},
	}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}

func TestGitmodulesBackslashBeforeTrailingSpace(t *testing.T) {
	content := "[submodule \"one\"]\n" +
		"\tpath = one\n" +
		"\turl = https://github.com/example/one.git\\ \n" +
		"\tbranch = main\n"
	res, err := (&gitmodulesParser{}).Parse(".gitmodules", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{{Name: "one", Scope: core.Runtime, Direct: true, NoPURL: true}}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}

func TestGitmodulesOtherSectionEndsSubmodule(t *testing.T) {
	content := "[submodule \"a\"]\n" +
		"\tpath = a\n" +
		"[foo]\n" +
		"\turl = https://github.com/example/other.git\n" +
		"[submodule \"b\"]\n" +
		"\tpath = b\n" +
		"\turl = https://github.com/example/b.git\n"
	res, err := (&gitmodulesParser{}).Parse(".gitmodules", []byte(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	want := []core.Dependency{
		{Name: "a", Scope: core.Runtime, Direct: true, NoPURL: true},
		{Name: "github.com/example/b", Scope: core.Runtime, Direct: true,
			Source: core.Source{Kind: core.SourceGit, Value: "https://github.com/example/b.git"}},
	}
	if !reflect.DeepEqual(res.Dependencies, want) {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}
