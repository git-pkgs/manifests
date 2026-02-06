package git

import (
	"os"
	"testing"
)

func TestGitmodules(t *testing.T) {
	content, err := os.ReadFile("../../testdata/git/.gitmodules")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gitmodulesParser{}
	deps, err := parser.Parse(".gitmodules", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]string)
	for _, d := range deps {
		depMap[d.Name] = d.RegistryURL
	}

	expected := map[string]string{
		"lib/foo":    "https://github.com/example/foo.git",
		"vendor/bar": "git@github.com:example/bar.git",
		"tools/baz":  "https://gitlab.com/example/baz.git",
	}

	for name, wantURL := range expected {
		gotURL, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if gotURL != wantURL {
			t.Errorf("%s url = %q, want %q", name, gotURL, wantURL)
		}
	}
}
