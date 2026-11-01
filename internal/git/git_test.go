package git

import (
	"os"
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
		if dep != want[i] {
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
	if len(res.Dependencies) != 1 || res.Dependencies[0] != want[0] {
		t.Errorf("got %+v, want %+v", res.Dependencies, want)
	}
}
