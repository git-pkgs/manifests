package julia

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestJuliaProject(t *testing.T) {
	content, err := os.ReadFile("../../testdata/julia/Project.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &juliaProjectParser{}
	deps, err := parser.Parse("Project.toml", content)
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

	// Check JSON with compat version
	if dep, ok := depMap["JSON"]; !ok {
		t.Error("expected JSON dependency")
	} else {
		if dep.Version != "0.21" {
			t.Errorf("expected JSON compat version 0.21, got %s", dep.Version)
		}
		if !dep.Direct {
			t.Error("expected direct dependency")
		}
	}

	// Check HTTP with compat version
	if dep, ok := depMap["HTTP"]; !ok {
		t.Error("expected HTTP dependency")
	} else if dep.Version != "1.0" {
		t.Errorf("expected HTTP compat version 1.0, got %s", dep.Version)
	}

	// Check Dates without compat version
	if dep, ok := depMap["Dates"]; !ok {
		t.Error("expected Dates dependency")
	} else if dep.Version != "" {
		t.Errorf("expected Dates version empty (no compat), got %s", dep.Version)
	}
}

func TestJuliaManifest(t *testing.T) {
	content, err := os.ReadFile("../../testdata/julia/Manifest.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &juliaManifestParser{}
	deps, err := parser.Parse("Manifest.toml", content)
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

	// Check Dates
	if dep, ok := depMap["Dates"]; !ok {
		t.Error("expected Dates dependency")
	} else if dep.Version != "1.8.5+0" {
		t.Errorf("expected Dates version 1.8.5+0, got %s", dep.Version)
	}

	// Check HTTP
	if dep, ok := depMap["HTTP"]; !ok {
		t.Error("expected HTTP dependency")
	} else if dep.Version != "1.5.0" {
		t.Errorf("expected HTTP version 1.5.0, got %s", dep.Version)
	}

	// Check JSON
	if dep, ok := depMap["JSON"]; !ok {
		t.Error("expected JSON dependency")
	} else if dep.Version != "0.21.4" {
		t.Errorf("expected JSON version 0.21.4, got %s", dep.Version)
	}
}

func TestJuliaRequire(t *testing.T) {
	content, err := os.ReadFile("../../testdata/julia/REQUIRE")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &juliaRequireParser{}
	deps, err := parser.Parse("REQUIRE", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check julia (base runtime requirement)
	if dep, ok := depMap["julia"]; !ok {
		t.Error("expected julia dependency")
	} else if dep.Version != "0.3" {
		t.Errorf("julia version = %q, want %q", dep.Version, "0.3")
	}

	// Check Colors with version
	if dep, ok := depMap["Colors"]; !ok {
		t.Error("expected Colors dependency")
	} else if dep.Version != "0.3.4" {
		t.Errorf("Colors version = %q, want %q", dep.Version, "0.3.4")
	}

	// Check Codecs without version
	if dep, ok := depMap["Codecs"]; !ok {
		t.Error("expected Codecs dependency")
	} else if dep.Version != "" {
		t.Errorf("Codecs version = %q, want empty", dep.Version)
	}

	// Check Plots with version range
	if dep, ok := depMap["Plots"]; !ok {
		t.Error("expected Plots dependency")
	} else if dep.Version != "0.12 0.15" {
		t.Errorf("Plots version = %q, want %q", dep.Version, "0.12 0.15")
	}

	// Check platform-specific dep (Homebrew from @osx)
	if _, ok := depMap["Homebrew"]; !ok {
		t.Error("expected Homebrew dependency (platform-specific)")
	}

	// Verify commented packages are excluded
	if _, ok := depMap["MySQL"]; ok {
		t.Error("MySQL should be excluded (in comment section)")
	}
}
