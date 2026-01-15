package cocoapods

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPodfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cocoapods/Podfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &podfileParser{}
	deps, err := parser.Parse("Podfile", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 31 {
		t.Fatalf("expected 31 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	expected := map[string]string{
		"Artsy+UIFonts":              "~> 1.1.0",
		"ISO8601DateFormatter":       "0.7",
		"ARCollectionViewMasonryLayout": "~> 2.0.0",
		"SDWebImage":                 "~> 3.7",
		"ReactiveCocoa":              "~> 4.0.1-alpha-2",
		"Nimble":                     "= 2.0.0-rc.3",
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

	// Verify some pods without versions exist
	if _, ok := depMap["Artsy+UIColors"]; !ok {
		t.Error("expected Artsy+UIColors dependency")
	}
	if _, ok := depMap["Quick"]; !ok {
		t.Error("expected Quick dependency")
	}
}

func TestPodfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cocoapods/Podfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &podfileLockParser{}
	deps, err := parser.Parse("Podfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 50 {
		t.Fatalf("expected 50 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	expected := map[string]string{
		"Alamofire":                  "2.0.1",
		"ARAnalytics/CoreIOS":        "3.8.0",
		"ARCollectionViewMasonryLayout": "2.0.0",
		"Artsy+UIColors":             "1.0.0",
		"Artsy+UIFonts":              "1.1.0",
		"CardFlight":                 "1.9.2",
		"FBSnapshotTestCase":         "1.8.1",
		"Forgeries":                  "0.1.0",
		"ISO8601DateFormatter":       "0.7",
		"Mixpanel":                   "2.8.3",
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

func TestPodspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/cocoapods/example.podspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &podspecParser{}
	deps, err := parser.Parse("example.podspec", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 4 {
		t.Fatalf("expected 4 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 4 packages (CocoaLumberjack subspecs)
	expectedNames := []string{
		"CocoaLumberjack/Core",
		"CocoaLumberjack/Default",
		"CocoaLumberjack/Extensions",
	}

	for _, name := range expectedNames {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
	}
}
