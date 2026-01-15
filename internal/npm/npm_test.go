package npm

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestNpmPackageJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/package.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageJSONParser{}
	deps, err := parser.Parse("package.json", content)
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

	// Check babel dependency (runtime)
	if babel, ok := depMap["babel"]; !ok {
		t.Error("expected babel dependency")
	} else {
		if babel.Version != "^4.6.6" {
			t.Errorf("babel version = %q, want %q", babel.Version, "^4.6.6")
		}
		if babel.Scope != core.Runtime {
			t.Errorf("babel scope = %q, want %q", babel.Scope, core.Runtime)
		}
		if !babel.Direct {
			t.Error("babel should be direct dependency")
		}
	}

	// Check mocha dependency (dev)
	if mocha, ok := depMap["mocha"]; !ok {
		t.Error("expected mocha dependency")
	} else {
		if mocha.Scope != core.Development {
			t.Errorf("mocha scope = %q, want %q", mocha.Scope, core.Development)
		}
	}

	// Check alias handling
	if actual, ok := depMap["@some-scope/actual-package"]; !ok {
		t.Error("expected aliased dependency @some-scope/actual-package")
	} else {
		if actual.Version != "^1.1.3" {
			t.Errorf("alias version = %q, want %q", actual.Version, "^1.1.3")
		}
	}

	// Verify comment was filtered out
	if _, ok := depMap["// my comment"]; ok {
		t.Error("comment should have been filtered out")
	}
}

func TestNpmPackageLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
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

	// Check express dependency
	if express, ok := depMap["express"]; !ok {
		t.Error("expected express dependency")
	} else {
		if express.Version != "4.15.3" {
			t.Errorf("express version = %q, want %q", express.Version, "4.15.3")
		}
		if express.Integrity != "sha1-urZdDwOqgMNYQIly/HAPkWlEtmI=" {
			t.Errorf("express integrity = %q", express.Integrity)
		}
	}

	// Check optional dependency
	if tweetnacl, ok := depMap["tweetnacl"]; ok {
		if tweetnacl.Scope != core.Optional {
			t.Errorf("tweetnacl scope = %q, want %q", tweetnacl.Scope, core.Optional)
		}
	}
}

func TestNpmPackageLockV3(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-lockfile-version-3/package-lock.json")
	if err != nil {
		t.Skipf("v3 fixture not found: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestParseNpmAlias(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		wantName string
		wantVer  string
	}{
		{"pkg", "1.0.0", "pkg", "1.0.0"},
		{"alias", "npm:real-pkg@1.0.0", "real-pkg", "1.0.0"},
		{"alias", "npm:@scope/pkg@^2.0.0", "@scope/pkg", "^2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name+"->"+tt.version, func(t *testing.T) {
			gotName, gotVer := parseNpmAlias(tt.name, tt.version)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotVer != tt.wantVer {
				t.Errorf("version = %q, want %q", gotVer, tt.wantVer)
			}
		})
	}
}

func TestExtractPackageName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"node_modules/express", "express"},
		{"node_modules/@types/node", "@types/node"},
		{"node_modules/a/node_modules/b", "b"},
		{"node_modules/@scope/pkg/node_modules/nested", "nested"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := extractPackageName(tt.path)
			if got != tt.want {
				t.Errorf("extractPackageName(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestYarnLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/yarn.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &yarnLockParser{}
	deps, err := parser.Parse("yarn.lock", content)
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

	// Check body-parser with resolved version
	if dep, ok := depMap["body-parser"]; !ok {
		t.Error("expected body-parser dependency")
	} else if dep.Version != "1.16.1" {
		t.Errorf("body-parser version = %q, want %q", dep.Version, "1.16.1")
	}

	// Check bytes
	if dep, ok := depMap["bytes"]; !ok {
		t.Error("expected bytes dependency")
	} else if dep.Version != "2.4.0" {
		t.Errorf("bytes version = %q, want %q", dep.Version, "2.4.0")
	}

	// Check yarn alias - parser captures alias name from header
	if dep, ok := depMap["alias-package-name"]; !ok {
		t.Error("expected alias-package-name dependency")
	} else if dep.Version != "1.1.3" {
		t.Errorf("alias version = %q, want %q", dep.Version, "1.1.3")
	}
}

func TestPnpmLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pnpmLockParser{}
	deps, err := parser.Parse("pnpm-lock.yaml", content)
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

	// Check chalk from pnpm v5 style
	if dep, ok := depMap["chalk"]; !ok {
		t.Error("expected chalk dependency")
	} else if dep.Version != "1.1.3" {
		t.Errorf("chalk version = %q, want %q", dep.Version, "1.1.3")
	}

	// Check ansi-styles dev dependency marking
	if dep, ok := depMap["ansi-styles"]; !ok {
		t.Error("expected ansi-styles dependency")
	} else if dep.Version == "" {
		t.Error("expected ansi-styles to have version")
	}

	// Check scoped package
	if dep, ok := depMap["@typescript-eslint/types"]; !ok {
		t.Error("expected @typescript-eslint/types dependency")
	} else if dep.Version != "5.13.0" {
		t.Errorf("@typescript-eslint/types version = %q, want %q", dep.Version, "5.13.0")
	}
}

func TestBunLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/bun.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &bunLockParser{}
	deps, err := parser.Parse("bun.lock", content)
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

	// Check zod (aliased as alias-package in workspace)
	if dep, ok := depMap["zod"]; !ok {
		t.Error("expected zod dependency")
	} else if dep.Version != "3.24.2" {
		t.Errorf("zod version = %q, want %q", dep.Version, "3.24.2")
	}

	// Check babel
	if dep, ok := depMap["babel"]; !ok {
		t.Error("expected babel dependency")
	} else if dep.Version != "6.23.0" {
		t.Errorf("babel version = %q, want %q", dep.Version, "6.23.0")
	}

	// Check integrity is captured
	if dep, ok := depMap["lodash"]; !ok {
		t.Error("expected lodash dependency")
	} else if dep.Integrity == "" {
		t.Error("expected lodash to have integrity")
	}

	// Check scoped package
	if dep, ok := depMap["@types/bun"]; !ok {
		t.Error("expected @types/bun dependency")
	} else if dep.Version != "1.2.5" {
		t.Errorf("@types/bun version = %q, want %q", dep.Version, "1.2.5")
	}
}

func TestNpmShrinkwrap(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-shrinkwrap.json")
	if err != nil {
		t.Skipf("shrinkwrap fixture not found: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("npm-shrinkwrap.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}
}

func TestParsePnpmPackageKey(t *testing.T) {
	tests := []struct {
		key      string
		wantName string
		wantVer  string
	}{
		{"/chalk/1.1.3", "chalk", "1.1.3"},
		{"/@scope/pkg/2.0.0", "@scope/pkg", "2.0.0"},
		{"chalk@1.1.3", "chalk", "1.1.3"},
		{"@scope/pkg@2.0.0", "@scope/pkg", "2.0.0"},
		{"@typescript-eslint/types@5.13.0", "@typescript-eslint/types", "5.13.0"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotName, gotVer := parsePnpmPackageKey(tt.key)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotVer != tt.wantVer {
				t.Errorf("version = %q, want %q", gotVer, tt.wantVer)
			}
		})
	}
}

func TestParseBunPackageKey(t *testing.T) {
	tests := []struct {
		key      string
		wantName string
		wantVer  string
	}{
		{"lodash@4.17.21", "lodash", "4.17.21"},
		{"@types/node@22.13.10", "@types/node", "22.13.10"},
		{"zod@3.24.2", "zod", "3.24.2"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			gotName, gotVer := parseBunPackageKey(tt.key)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotVer != tt.wantVer {
				t.Errorf("version = %q, want %q", gotVer, tt.wantVer)
			}
		})
	}
}
