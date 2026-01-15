package carthage

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestCartfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/carthage/Cartfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cartfileParser{}
	deps, err := parser.Parse("Cartfile", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 8 {
		t.Fatalf("expected 8 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	expected := map[string]string{
		"ReactiveCocoa/ReactiveCocoa":  ">=2.3.1",
		"Mantle/Mantle":                "~>1.0",
		"jspahrsummers/libextobjc":     "==0.4.1",
	}

	for name, wantVer := range expected {
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

func TestCartfilePrivate(t *testing.T) {
	content, err := os.ReadFile("../../testdata/carthage/Cartfile.private")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cartfileParser{}
	deps, err := parser.Parse("Cartfile.private", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 3 packages
	expected := map[string]string{
		"Quick/Quick":            "~>0.9",
		"Quick/Nimble":           "~>3.1",
		"jspahrsummers/xcconfigs": "ec5753493605deed7358dec5f9260f503d3ed650",
	}

	for name, wantVer := range expected {
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

func TestCartfileResolved(t *testing.T) {
	content, err := os.ReadFile("../../testdata/carthage/Cartfile.resolved")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &cartfileResolvedParser{}
	deps, err := parser.Parse("Cartfile.resolved", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 9 {
		t.Fatalf("expected 9 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 9 packages with versions
	expected := map[string]string{
		"thoughtbot/Argo":           "v2.2.0",
		"Quick/Nimble":              "v3.1.0",
		"jdhealy/PrettyColors":      "v3.0.0",
		"Quick/Quick":               "v0.9.1",
		"antitypical/Result":        "1.0.2",
		"jspahrsummers/xcconfigs":   "ec5753493605deed7358dec5f9260f503d3ed650",
		"Carthage/Commandant":       "0.8.3",
		"ReactiveCocoa/ReactiveCocoa": "v4.0.1",
		"Carthage/ReactiveTask":     "0.9.1",
	}

	for name, wantVer := range expected {
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
