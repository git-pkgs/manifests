package asdf

import (
	"os"
	"testing"
)

func TestToolVersions(t *testing.T) {
	content, err := os.ReadFile("../../testdata/asdf/.tool-versions")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &toolVersionsParser{}
	res, err := parser.Parse(".tool-versions", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(res.Dependencies) != 6 {
		t.Fatalf("expected 6 dependencies, got %d", len(res.Dependencies))
	}

	depMap := make(map[string]string)
	for _, d := range res.Dependencies {
		depMap[d.Name] = d.Version
	}

	expected := map[string]string{
		"nodejs": "18.12.0",
		"ruby":   "3.2.1",
		"python": "3.11.2",
		"golang": "1.21.0",
		"erlang": "26.0.2",
		"rust":   "1.72.0",
	}

	for name, wantVer := range expected {
		gotVer, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if gotVer != wantVer {
			t.Errorf("%s version = %q, want %q", name, gotVer, wantVer)
		}
	}
}
