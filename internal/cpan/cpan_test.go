package cpan

import (
	"os"
	"testing"
)

func TestCpanfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cpan/cpanfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cpanfileParser{}
	deps, err := parser.Parse("cpanfile", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestCpanfileSnapshot(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cpan/cpanfile.snapshot")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cpanfileSnapshotParser{}
	deps, err := parser.Parse("cpanfile.snapshot", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestMakefilePL(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cpan/Makefile.PL")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &makefilePLParser{}
	deps, err := parser.Parse("Makefile.PL", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestMetaJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cpan/META.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &metaJSONParser{}
	deps, err := parser.Parse("META.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestMetaYML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cpan/META.yml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &metaYMLParser{}
	deps, err := parser.Parse("META.yml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}
