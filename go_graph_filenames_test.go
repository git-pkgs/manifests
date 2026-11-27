package manifests

import (
	"os"
	"reflect"
	"testing"
)

func TestGoGraphAlternateFilenames(t *testing.T) {
	content, err := os.ReadFile("testdata/golang/go.graph")
	if err != nil {
		t.Fatal(err)
	}
	want, err := Parse("go.graph", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.Dependencies) == 0 {
		t.Fatal("fixture has no dependencies")
	}
	for _, name := range []string{"go.mod.graph", "go.modgraph", "nested/dir/go.mod.graph"} {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(name, content)
			if err != nil {
				t.Fatal(err)
			}
			if got.Ecosystem != "golang" || got.Kind != Lockfile {
				t.Fatalf("identified as %s %s", got.Ecosystem, got.Kind)
			}
			if !reflect.DeepEqual(got.Dependencies, want.Dependencies) {
				t.Errorf("dependencies differ from go.graph")
			}
		})
	}
}

func TestGoGraphSimilarNamesNotMatched(t *testing.T) {
	for _, name := range []string{"go.mod.graph.txt", "go.graph.bak", "my-go.modgraph"} {
		if _, _, ok := Identify(name); ok {
			t.Errorf("%s should not be identified", name)
		}
	}
}
