package cpan

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
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

	if len(deps) != 108 {
		t.Fatalf("expected 108 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"List::MoreUtils":    "0.402",
		"Guard":              "1.023",
		"PadWalker":          "2.2",
		"aliased":            "0.34",
		"Catalyst":           "5.80031",
		"ExtUtils::MakeMaker": "6.72",
	}

	for name, wantVer := range samples {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != wantVer {
			t.Errorf("%s version = %q, want %q", name, dep.Version, wantVer)
		}
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

	if len(deps) != 2898 {
		t.Fatalf("expected 2898 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Verify some packages exist
	sampleNames := []string{"Moose", "Catalyst", "DBI", "Test::More"}
	for _, name := range sampleNames {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
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

	if len(deps) != 11 {
		t.Fatalf("expected 11 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 11 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"Moo", "2.0", core.Runtime},
		{"JSON::XS", "0", core.Runtime},
		{"LWP::UserAgent", "6.0", core.Runtime},
		{"URI", "0", core.Runtime},
		{"Data::Dumper", "0", core.Runtime},
		{"File::Spec", "0.8", core.Runtime},
		{"File::Temp", "0", core.Build},
		{"Test::More", "0.88", core.Test},
		{"Test::Deep", "0", core.Test},
		{"Test::Warn", "0.30", core.Test},
		{"ExtUtils::MakeMaker", "6.64", core.Build},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
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

	if len(deps) != 5 {
		t.Fatalf("expected 5 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 5 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"Getopt::Long", "2.32", core.Runtime},
		{"List::Util", "1.07_00", core.Runtime},
		{"English", "1.00", core.Build},
		{"Test::More", "0.45", core.Build},
		{"Module::Build", "0.28", core.Build},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
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

	if len(deps) != 10 {
		t.Fatalf("expected 10 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 10 packages with scopes
	expected := []struct {
		name  string
		scope core.Scope
	}{
		{"File::Temp", core.Runtime},
		{"LWP", core.Runtime},
		{"XML::Simple", core.Runtime},
		{"Digest::MD5", core.Runtime},
		{"ExtUtils::MakeMaker", core.Build},
		{"Net::IP", core.Optional},
		{"Nvidia::ml", core.Optional},
		{"Proc::Daemon", core.Optional},
		{"Proc::PID::File", core.Optional},
		{"Compress::Zlib", core.Optional},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}

func TestBuildPL(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cpan/Build.PL")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &buildPLParser{}
	deps, err := parser.Parse("Build.PL", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 11 {
		t.Fatalf("expected 11 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 11 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"File::Temp", "0", core.Build},
		{"File::Spec", "0", core.Build},
		{"Test::More", "0.96", core.Test},
		{"Test::Fatal", "0", core.Test},
		{"Test::Warn", "0", core.Test},
		{"Module::Build", "0.40", core.Build},
		{"Moose", "2.0", core.Runtime},
		{"namespace::clean", "0.20", core.Runtime},
		{"Try::Tiny", "0.22", core.Runtime},
		{"JSON", "0", core.Runtime},
		{"YAML", "0", core.Runtime},
	}

	for _, exp := range expected {
		dep, ok := depMap[exp.name]
		if !ok {
			t.Errorf("expected %s dependency", exp.name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", exp.name, dep.Version, exp.version)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}
