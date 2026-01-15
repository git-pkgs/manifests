package vcpkg

import (
	"os"
	"testing"
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]bool)
	for _, d := range deps {
		depMap[d.Name] = true
	}

	// Check string dependencies
	expectedDeps := []string{
		"sdl2", "physfs", "harfbuzz", "fribidi", "libogg",
		"libtheora", "libvorbis", "opus", "libpng", "freetype",
		"gettext", "openal-soft", "zlib", "sqlite3", "libsodium",
	}

	for _, name := range expectedDeps {
		if !depMap[name] {
			t.Errorf("expected %s dependency", name)
		}
	}

	// Check object dependencies (curl, angle, basisu)
	if !depMap["curl"] {
		t.Error("expected curl dependency")
	}
	if !depMap["angle"] {
		t.Error("expected angle dependency")
	}
	if !depMap["basisu"] {
		t.Error("expected basisu dependency")
	}
}
