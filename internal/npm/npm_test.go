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

	// package-lock.json has many dependencies
	if len(deps) < 100 {
		t.Fatalf("expected at least 100 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with exact versions and integrities
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"express":        {"4.15.3", "sha1-urZdDwOqgMNYQIly/HAPkWlEtmI="},
		"accepts":        {"1.3.3", "sha1-w8p0NJOGSMPg2cHjKN1otiLChMo="},
		"ajv":            {"4.11.8", "sha1-gv+wKynmYq5TvcIK8VlHcGc5xTY="},
		"ansi-regex":     {"2.1.1", "sha1-w7M6te42DYbg5ijwRorn7yfWVN8="},
		"ansi-styles":    {"2.2.1", "sha1-tDLdM1i2NM914eRmQ2gkBTPB3b4="},
		"body-parser":    {"1.17.2", "sha1-+IkqvI+eYn1Crtr7yma/WrmRBO4="},
		"bytes":          {"2.4.0", "sha1-fZcZb51br39pNeJZhVSe3SpsIzk="},
		"chalk":          {"1.1.3", "sha1-qBFcVeSnAv5NFQq9OHKCKn4J/Jg="},
		"content-type":   {"1.0.2", "sha1-t9ETrueo3Se9IRM8TcJSnfFyHu0="},
		"cookie":         {"0.3.1", "sha1-5+Ch+e9DtMi6klxcWpboBtFoc7s="},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, exp.integrity)
		}
	}

	// Check optional dependency
	if tweetnacl, ok := depMap["tweetnacl"]; ok {
		if tweetnacl.Scope != core.Optional {
			t.Errorf("tweetnacl scope = %v, want %v", tweetnacl.Scope, core.Optional)
		}
	}
}

func TestNpmPackageLockV1(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-lockfile-version-1/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
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

	// All 3 packages (semver-regex appears twice with different versions)
	expected := []string{"find-versions", "semver-regex"}
	for _, name := range expected {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
	}
}

func TestNpmPackageLockV2(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-lockfile-version-2/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
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

	// All 3 packages (semver-regex appears twice with different versions)
	expected := []string{"find-versions", "semver-regex"}
	for _, name := range expected {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
	}
}

func TestNpmPackageLockV3(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-lockfile-version-3/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
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

	// All 4 packages (alias-package-name, find-versions, semver-regex x2)
	expected := []string{"alias-package-name", "find-versions", "semver-regex"}
	for _, name := range expected {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
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

	if len(deps) != 21 {
		t.Fatalf("expected 21 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 21 packages with exact versions
	expected := map[string]string{
		"body-parser":        "1.16.1",
		"bytes":              "2.4.0",
		"content-type":       "1.0.2",
		"debug":              "2.6.1",
		"depd":               "1.1.0",
		"ee-first":           "1.1.1",
		"http-errors":        "1.5.1",
		"iconv-lite":         "0.4.15",
		"inherits":           "2.0.3",
		"media-typer":        "0.3.0",
		"mime-db":            "1.26.0",
		"mime-types":         "2.1.14",
		"ms":                 "0.7.2",
		"on-finished":        "2.3.0",
		"qs":                 "6.2.1",
		"raw-body":           "2.2.0",
		"setprototypeof":     "1.0.2",
		"statuses":           "1.3.1",
		"alias-package-name": "1.1.3", // aliased from @some-scope/actual-package
		"type-is":            "1.6.14",
		"unpipe":             "1.0.0",
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

	// pnpm-lock.yaml has 8 unique packages (ansi-styles has 2 versions but may be deduped)
	if len(deps) != 8 {
		t.Fatalf("expected 8 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 8 packages with exact versions and integrities
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"ansi-regex":               {"2.1.1", "sha1-w7M6te42DYbg5ijwRorn7yfWVN8="},
		"chalk":                    {"1.1.3", "sha1-qBFcVeSnAv5NFQq9OHKCKn4J/Jg="},
		"escape-string-regexp":     {"1.0.5", "sha1-G2HAViGQqN/2rjuyzwIAyhMLhtQ="},
		"has-ansi":                 {"2.0.0", "sha1-NPUEnOHs3ysGSa8+8k5F7TVBbZE="},
		"strip-ansi":               {"3.0.1", "sha1-ajhfuIU9lS1f8F0Oiq+UJ43GPc8="},
		"supports-color":           {"2.0.0", "sha1-U10EXOa2Nj+kARcIRimZXp3zJMc="},
		"@typescript-eslint/types": {"5.13.0", "sha512-LmE/KO6DUy0nFY/OoQU0XelnmDt+V8lPQhh8MOVa7Y5k2gGRd6U9Kp3wAjhB4OHg57tUO0nOnwYQhRRyEAyOyg=="},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, exp.integrity)
		}
	}

	// ansi-styles has 2 versions (2.2.0 dev:false, 2.2.1 dev:true) - verify one exists
	if dep, ok := depMap["ansi-styles"]; !ok {
		t.Error("expected ansi-styles dependency")
	} else if dep.Version == "" {
		t.Error("expected ansi-styles to have version")
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

	// isarray is a file: dependency with no version, may be skipped
	if len(deps) < 10 {
		t.Fatalf("expected at least 10 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All packages with exact versions and integrities
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"@types/bun":   {"1.2.5", "sha512-w2OZTzrZTVtbnJew1pdFmgV99H0/L+Pvw+z1P67HaR18MHOzYnTYOi6qzErhK8HyT+DB782ADVPPE92Xu2/Opg=="},
		"@types/node":  {"22.13.10", "sha512-I6LPUvlRH+O6VRUqYOcMudhaIdUVWfsjnZavnsraHvpBwaEyMN29ry+0UVJhImYL16xsscu0aske3yA+uPOWfw=="},
		"@types/ws":    {"8.5.14", "sha512-bd/YFLW+URhBzMXurx7lWByOu+xzU9+kb3RboOteXYDfW+tr+JZa99OyNmPINEGB/ahzKrEuc8rcv4gnpJmxTw=="},
		"zod":          {"3.24.2", "sha512-lY7CDW43ECgW9u1TcT3IoXHflywfVqDYze4waEz812jR/bZ8FHDsl7pFQoSZTz5N+2NqRXs8GBwnAwo3ZNxqhQ=="},
		"babel":        {"6.23.0", "sha512-ZDcCaI8Vlct8PJ3DvmyqUz+5X2Ylz3ZuuItBe/74yXosk2dwyVo/aN7MCJ8HJzhnnJ+6yP4o+lDgG9MBe91DLA=="},
		"bun-types":    {"1.2.5", "sha512-3oO6LVGGRRKI4kHINx5PIdIgnLRb7l/SprhzqXapmoYkFl5m4j6EvALvbDVuuBFaamB46Ap6HCUxIXNLCGy+tg=="},
		"lodash":       {"4.17.21", "sha512-v2kDEe57lecTulaDIuNTPy3Ry4gLGJ6Z1O3vE1krgXZNrsQ+LFTGHVxVjcXPs17LhbZVGedAJv8XZ1tvj5FvSg=="},
		"prettier":     {"3.5.3", "sha512-QQtaxnoDJeAkDvDKWCLiwIXkTgRhwYDEQCghU9Z6q03iyek/rxRh/2lC3HB7P8sWT2xC/y5JDctPLBIGzHKbhw=="},
		"typescript":   {"5.8.2", "sha512-aJn6wq13/afZp/jT9QZmwEjDqqvSGp1VT5GVg+f/t6/oVyrgXM6BY1h9BRh/O5p3PlUPAe+WuiEZOmb/49RqoQ=="},
		"undici-types": {"6.20.0", "sha512-Ny6QZ2Nju20vw1SRHe3d9jVu6gJ+4e3+MMpqu7pqE5HT6WsTSlce++GQmK5UXS8mzV8DSYHrQH+Xrf2jVcuKNg=="},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, exp.integrity)
		}
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

func Test2018PackageLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/2018-package-lock/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1974 {
		t.Fatalf("expected 1974 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages
	samples := []string{
		"babel-plugin-transform-es2015-literals",
		"evp_bytestokey",
		"form-data",
		"html-comment-regex",
		"isomorphic-fetch",
		"json5",
		"os-tmpdir",
		"array-includes",
	}

	for _, name := range samples {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
	}
}

func TestNpmLocalFilePackageLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-local-file/package-lock.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmPackageLockParser{}
	deps, err := parser.Parse("package-lock.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 7 {
		t.Fatalf("expected 7 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 7 packages
	expected := map[string]string{
		"react":        "18.3.1",
		"src":          "1.0.0",
		"js-tokens":    "4.0.0",
		"left-pad":     "1.3.0",
		"lodash":       "4.17.21",
		"loose-envify": "1.4.0",
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

	// other-package has no version (file: dependency)
	if dep, ok := depMap["other-package"]; !ok {
		t.Error("expected other-package dependency")
	} else if dep.Version != "" {
		t.Errorf("other-package version = %q, want empty", dep.Version)
	}
}

func TestNpmLocalFileYarnLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-local-file/yarn.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &yarnLockParser{}
	deps, err := parser.Parse("yarn.lock", content)
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

	// All 5 packages
	expected := map[string]string{
		"js-tokens":     "4.0.0",
		"left-pad":      "1.3.0",
		"loose-envify":  "1.4.0",
		"other-package": "1.0.0",
		"react":         "18.3.1",
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

func TestPnpmLockV5(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/pnpm-lockfile-version-5/pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pnpmLockParser{}
	deps, err := parser.Parse("pnpm-lock.yaml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 89 {
		t.Fatalf("expected 89 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages
	samples := map[string]string{
		"@babel/helper-string-parser":      "7.27.1",
		"@babel/helper-validator-identifier": "7.27.1",
		"@babel/types":                     "7.28.1",
		"acorn-babel":                      "0.11.1-38",
		"acorn":                            "5.7.4",
		"amdefine":                         "1.0.1",
		"ansi-regex":                       "2.1.1",
		"ansi-styles":                      "2.2.1",
		"babel":                            "4.7.16",
		"chalk":                            "1.1.3",
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

func TestPnpmLockV6(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/pnpm-lockfile-version-6/pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pnpmLockParser{}
	deps, err := parser.Parse("pnpm-lock.yaml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 90 {
		t.Fatalf("expected 90 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages
	samples := map[string]string{
		"@babel/helper-string-parser":      "7.27.1",
		"@babel/helper-validator-identifier": "7.27.1",
		"@babel/types":                     "7.28.1",
		"acorn-babel":                      "0.11.1-38",
		"acorn":                            "5.7.4",
		"amdefine":                         "1.0.1",
		"brace-expansion":                  "1.1.12",
		"chalk":                            "1.1.3",
		"chokidar":                         "0.12.6",
		"commander":                        "0.6.1",
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

func TestPnpmLockV9(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/pnpm-lockfile-version-9/pnpm-lock.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pnpmLockParser{}
	deps, err := parser.Parse("pnpm-lock.yaml", content)
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
		"@babel/helper-string-parser":      "7.27.1",
		"@babel/helper-validator-identifier": "7.27.1",
		"@babel/types":                     "7.28.1",
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

func TestYarnWithGitRepo(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/yarn-with-git-repo/yarn.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &yarnLockParser{}
	deps, err := parser.Parse("yarn.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	if deps[0].Name != "vue" {
		t.Errorf("expected vue, got %s", deps[0].Name)
	}
	if deps[0].Version != "2.6.12" {
		t.Errorf("vue version = %q, want %q", deps[0].Version, "2.6.12")
	}
}

func TestBowerJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/bower.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &bowerParser{}
	deps, err := parser.Parse("bower.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	if deps[0].Name != "jquery" {
		t.Errorf("expected jquery, got %s", deps[0].Name)
	}
	if deps[0].Version != ">= 1.9.1" {
		t.Errorf("jquery version = %q, want %q", deps[0].Version, ">= 1.9.1")
	}
}

func TestDenoJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/deno.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &denoJSONParser{}
	deps, err := parser.Parse("deno.json", content)
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

	// All 4 packages
	expected := map[string]string{
		"chalk":      "5.3.0",
		"lodash":     "",
		"@std/path":  "^1.0.0",
		"@std/fs":    "",
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

func TestDenoLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/deno.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &denoLockParser{}
	deps, err := parser.Parse("deno.lock", content)
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

	// All 4 packages with versions and integrities
	expected := []struct {
		name      string
		version   string
		integrity string
	}{
		{"@std/fs", "1.0.3", "sha256-abc123"},
		{"@std/path", "1.0.6", "sha256-def456"},
		{"chalk", "5.3.0", "sha512-xyz789"},
		{"lodash", "4.17.21", "sha512-uvw012"},
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
		if dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", exp.name, dep.Integrity, exp.integrity)
		}
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

func TestNpmLsJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/npm-ls.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &npmLsParser{}
	deps, err := parser.Parse("npm-ls.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 6 {
		t.Fatalf("expected 6 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All packages with expected versions
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"ansicolor":      {"1.1.93", ""},
		"babel-cli":      {"6.26.0", ""},
		"debug":          {"2.6.9", "sha512-bC7ElrdJaJnPbAP+1EotYvqZsb3ecl5wi6Bfi6BJTUcNowp6cvspg0jXznRTKDjm/E7AdgFBVeAPVMNcKGsHMA=="},
		"babel-polyfill": {"6.26.0", ""},
		"core-js":        {"2.6.12", ""},
		"lodash":         {"4.17.21", ""},
	}

	for name, exp := range expected {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if exp.version != "" && dep.Version != exp.version {
			t.Errorf("%s version = %q, want %q", name, dep.Version, exp.version)
		}
		if exp.integrity != "" && dep.Integrity != exp.integrity {
			t.Errorf("%s integrity = %q, want %q", name, dep.Integrity, exp.integrity)
		}
	}
}

func TestYarnV4Lock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/npm/yarn-v4-lockfile/yarn.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &yarnLockParser{}
	deps, err := parser.Parse("yarn.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 6 packages (excluding workspace)
	if len(deps) != 6 {
		t.Fatalf("expected 6 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All packages with expected versions
	expected := map[string]string{
		"js-tokens":       "4.0.0",
		"left-pad":        "1.3.0",
		"loose-envify":    "1.4.0",
		"react":           "18.3.1",
		"fsevents":        "2.3.2",
		"strip-ansi-cjs":  "6.0.1",
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

	// Check that workspace was excluded
	if _, ok := depMap["yarn-lock"]; ok {
		t.Error("workspace package should be excluded")
	}
}
