package vcpkg

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestVcpkgJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/vcpkg/vcpkg.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &vcpkgJSONParser{}
	deps, err := parser.Parse("vcpkg.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 18 {
		t.Fatalf("expected 18 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 18 packages
	expectedDeps := []string{
		"sdl2", "physfs", "harfbuzz", "fribidi", "libogg",
		"libtheora", "libvorbis", "opus", "libpng", "freetype",
		"gettext", "openal-soft", "zlib", "sqlite3", "libsodium",
		"curl", "angle", "basisu",
	}

	for _, name := range expectedDeps {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
	}
}
