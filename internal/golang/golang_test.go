package golang

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGoMod(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/go.mod")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &goModParser{}
	deps, err := parser.Parse("go.mod", content)
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

	// Check indirect dependency
	if redigo, ok := depMap["github.com/gomodule/redigo"]; !ok {
		t.Error("expected github.com/gomodule/redigo dependency")
	} else {
		if redigo.Version != "v2.0.0+incompatible" {
			t.Errorf("redigo version = %q, want %q", redigo.Version, "v2.0.0+incompatible")
		}
		if redigo.Direct {
			t.Error("redigo should be indirect dependency")
		}
	}

	// Check direct dependency (no // indirect comment)
	if yaml, ok := depMap["gopkg.in/yaml.v1"]; !ok {
		t.Error("expected gopkg.in/yaml.v1 dependency")
	} else {
		if yaml.Version != "v1.0.0-20140924161607-9f9df34309c0" {
			t.Errorf("yaml version = %q, want %q", yaml.Version, "v1.0.0-20140924161607-9f9df34309c0")
		}
		if !yaml.Direct {
			t.Error("yaml should be direct dependency")
		}
	}

	// Check single-line require
	if net, ok := depMap["golang.org/x/net"]; !ok {
		t.Error("expected golang.org/x/net dependency")
	} else {
		if net.Version != "v1.2.3" {
			t.Errorf("net version = %q, want %q", net.Version, "v1.2.3")
		}
	}

	// Check tool dependencies are marked as Development scope
	if report, ok := depMap["github.com/jstemmer/go-junit-report"]; !ok {
		t.Error("expected github.com/jstemmer/go-junit-report dependency")
	} else {
		if report.Scope != core.Development {
			t.Errorf("go-junit-report scope = %v, want Development", report.Scope)
		}
	}

	if reportV2, ok := depMap["github.com/jstemmer/go-junit-report/v2"]; !ok {
		t.Error("expected github.com/jstemmer/go-junit-report/v2 dependency")
	} else {
		if reportV2.Scope != core.Development {
			t.Errorf("go-junit-report/v2 scope = %v, want Development", reportV2.Scope)
		}
	}

	// Non-tool dependencies should be Runtime scope
	if depMap["github.com/gomodule/redigo"].Scope != core.Runtime {
		t.Errorf("redigo scope = %v, want Runtime", depMap["github.com/gomodule/redigo"].Scope)
	}
}

func TestGoSum(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/go.sum")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &goSumParser{}
	deps, err := parser.Parse("go.sum", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// go.sum has 6 unique packages with h1: hashes
	// kr/pty only has /go.mod entry (no h1: hash) so it's excluded
	if len(deps) != 6 {
		t.Fatalf("expected 6 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Verify all packages with exact versions and integrities
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"github.com/go-check/check":        {"v0.0.0-20180628173108-788fd7840127", "h1:0gkP6mzaMqkmpcJYCFOLkIBwI7xFExG03bbkOkCvUPI="},
		"github.com/gomodule/redigo":       {"v2.0.0+incompatible", "h1:K/R+8tc58AaqLkqG2Ol3Qk+DR/TlNuhuh457pBFPtt0="},
		"github.com/kr/pretty":             {"v0.1.0", "h1:L/CwN0zerZDmRFUapSPitk6f+Q3+0za1rQkzVuMiMFI="},
		"github.com/kr/text":               {"v0.1.0", "h1:45sCR5RtlFHMR4UwH9sdQ5TC8v0qDQCHnXt+kaKSTVE="},
		"github.com/replicon/fast-archiver": {"v0.0.0-20121220195659-060bf9adec25", "h1:aq3XSz9htmdvrxpK6eBIbjs3SaN8G1D9RuKkDo4PRnw="},
		"gopkg.in/yaml.v1":                 {"v1.0.0-20140924161607-9f9df34309c0", "h1:POO/ycCATvegFmVuPpQzZFJ+pGZeX22Ufu6fibxDVjU="},
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

func TestGodepsJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Godeps.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &godepsJSONParser{}
	deps, err := parser.Parse("Godeps.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 14 {
		t.Fatalf("expected 14 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 14 packages with expected versions (Comment if present, else Rev)
	expected := map[string]string{
		"github.com/BurntSushi/toml":             "v0.1.0-9-g3883ac1",
		"github.com/Sirupsen/logrus":             "v0.8.7",
		"github.com/ayufan/golang-kardianos-service": "9ce7ccf10c81705a8880170bbf506bd539bc69b2",
		"github.com/codegangsta/cli":             "1.2.0-139-g142e6cd",
		"github.com/fsouza/go-dockerclient":      "163268693e2cf8be2920158b59ef438fc77b85e2",
		"github.com/golang/mock/gomock":          "06883d979f10cc178f2716846215c8cf90f9e363",
		"github.com/kardianos/osext":             "efacde03154693404c65e7aa7d461ac9014acd0c",
		"github.com/ramr/go-reaper":              "1a6cbc07ef2f7e248769ef4efd80aaa16f97ec12",
		"github.com/stretchr/objx":               "cbeaeb16a013161a98496fad62933b1d21786672",
		"github.com/stretchr/testify/assert":     "1297dc01ed0a819ff634c89707081a4df43baf6b",
		"github.com/stretchr/testify/mock":       "1297dc01ed0a819ff634c89707081a4df43baf6b",
		"gitlab.com/ayufan/golang-cli-helpers":   "0a14b63a7466ee44de4a90f998fad73afa8482bf",
		"golang.org/x/crypto/ssh":                "1351f936d976c60a0a48d728281922cf63eafb8d",
		"gopkg.in/yaml.v1":                       "9f9df34309c04878acc86042b16630b0f696e1de",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
	}
}

func TestGlideYAML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/glide.yaml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &glideYAMLParser{}
	deps, err := parser.Parse("glide.yaml", content)
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

	// All 4 packages with expected versions
	expected := map[string]string{
		"gopkg.in/yaml.v2":            "",       // no version specified
		"github.com/Masterminds/vcs":  "^1.4.0",
		"github.com/codegangsta/cli":  "",       // no version specified
		"github.com/Masterminds/semver": "^1.0.0",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
		if !dep.Direct {
			t.Errorf("%s should be direct dependency", name)
		}
	}
}

func TestGlideLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/glide.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &glideLockParser{}
	deps, err := parser.Parse("glide.lock", content)
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

	// All 4 packages with exact commit hashes
	expected := map[string]string{
		"github.com/codegangsta/cli":    "c31a7975863e7810c92e2e288a9ab074f9a88f29",
		"github.com/Masterminds/semver": "513f3dcb3ecfb1248831fb5cb06a23a3cd5935dc",
		"github.com/Masterminds/vcs":    "9c0db6583837118d5df7c2ae38ab1c194e434b35",
		"gopkg.in/yaml.v2":              "f7716cbe52baa25d2e9b0d0da546fcf909fc16b4",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
	}
}

func TestGopkgTOML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Gopkg.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gopkgTOMLParser{}
	deps, err := parser.Parse("Gopkg.toml", content)
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

	// All 8 packages with expected versions (version or branch)
	expected := map[string]string{
		"github.com/Masterminds/semver":  "parse-constraints-with-dash-in-pre", // branch
		"github.com/Masterminds/vcs":     "1.11.0",
		"github.com/go-yaml/yaml":        "v2",     // branch
		"github.com/pelletier/go-toml":   "master", // branch
		"github.com/pkg/errors":          "0.8.0",
		"github.com/boltdb/bolt":         "1.0.0",
		"github.com/jmank88/nuts":        "0.2.0",
		"github.com/golang/protobuf":     "master", // branch
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
		if !dep.Direct {
			t.Errorf("%s should be direct dependency", name)
		}
	}
}

func TestGopkgLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Gopkg.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gopkgLockParser{}
	deps, err := parser.Parse("Gopkg.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 14 {
		t.Fatalf("expected 14 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 14 packages with expected versions (version tag if present, else revision)
	expected := map[string]string{
		"github.com/Masterminds/semver":  "a93e51b5a57ef416dac8bb02d11407b6f55d8929", // no version, use revision
		"github.com/Masterminds/vcs":     "v1.11.1",
		"github.com/armon/go-radix":      "4239b77079c7b5d1243b7b4736304ce8ddb6f0f2",
		"github.com/boltdb/bolt":         "v1.3.1",
		"github.com/go-yaml/yaml":        "cd8b52f8269e0feb286dfeef29f8fe4d5b397e0b",
		"github.com/golang/protobuf":     "5afd06f9d81a86d6e3bb7dc702d6bd148ea3ff23",
		"github.com/jmank88/nuts":        "v0.2.0",
		"github.com/nightlyone/lockfile": "e83dc5e7bba095e8d32fb2124714bf41f2a30cb5",
		"github.com/pelletier/go-toml":   "b8b5e7696574464b2f9bf303a7b37781bb52889f",
		"github.com/pkg/errors":          "v0.8.0",
		"github.com/sdboyer/constext":    "836a144573533ea4da4e6929c235fd348aed1c80",
		"golang.org/x/net":               "66aacef3dd8a676686c7ae3716979581e8b03c47",
		"golang.org/x/sync":              "f52d1811a62927559de87708c8913c1650ce4f26",
		"golang.org/x/sys":               "bb24a47a89eac6c1227fbcb2ae37a8b9ed323366",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
	}
}

func TestVendorJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/vendor.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &vendorJSONParser{}
	deps, err := parser.Parse("vendor.json", content)
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

	// All 5 packages with exact revisions
	// Note: golang.org/x/tools/go/vcs is truncated to golang.org/x/tools by extractBasePackage
	expected := map[string]string{
		"github.com/Bowery/prompt":   "d43c2707a6c5a152a344c64bb4fed657e2908a81",
		"github.com/dchest/safefile": "855e8d98f1852d48dde521e0522408d1fe7e836a",
		"github.com/google/shlex":    "6f45313302b9c56850fc17f99e40caebce98c716",
		"github.com/pkg/errors":      "a2d6902c6d2a2f194eb3fb474981ab7867c81505",
		"golang.org/x/tools":         "1727758746e7a08feaaceb9366d1468498ac2ac2",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
	}
}

func TestGoSingleRequireMod(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/go.single-require.mod")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &goModParser{}
	deps, err := parser.Parse("go.mod", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.Name != "github.com/go-check/check" {
		t.Errorf("name = %q, want %q", dep.Name, "github.com/go-check/check")
	}
	if dep.Version != "v0.0.0-20180628173108-788fd7840127" {
		t.Errorf("version = %q, want %q", dep.Version, "v0.0.0-20180628173108-788fd7840127")
	}
	if dep.Direct {
		t.Error("expected indirect dependency")
	}
}

func TestGoModToolDependencies(t *testing.T) {
	// Test that tool dependencies are marked as Development scope
	content := []byte(`module test

go 1.24

require (
	example.com/runtime-pkg v1.0.0
	example.com/tool-pkg v2.0.0
	golang.org/x/tools v0.20.0
)

tool example.com/tool-pkg/cmd/mytool

tool (
	golang.org/x/tools/cmd/stringer
)
`)

	parser := &goModParser{}
	deps, err := parser.Parse("go.mod", content)
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

	// Runtime dependency should be Runtime scope
	if runtime, ok := depMap["example.com/runtime-pkg"]; !ok {
		t.Error("expected example.com/runtime-pkg dependency")
	} else if runtime.Scope != core.Runtime {
		t.Errorf("runtime-pkg scope = %v, want Runtime", runtime.Scope)
	}

	// Tool dependency (exact match) should be Development scope
	if tool, ok := depMap["example.com/tool-pkg"]; !ok {
		t.Error("expected example.com/tool-pkg dependency")
	} else if tool.Scope != core.Development {
		t.Errorf("tool-pkg scope = %v, want Development", tool.Scope)
	}

	// Tool dependency (module is prefix of tool path) should be Development scope
	if tools, ok := depMap["golang.org/x/tools"]; !ok {
		t.Error("expected golang.org/x/tools dependency")
	} else if tools.Scope != core.Development {
		t.Errorf("golang.org/x/tools scope = %v, want Development", tools.Scope)
	}
}

func TestGoResolvedDepsJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/go-resolved-dependencies.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &goResolvedDepsParser{}
	deps, err := parser.Parse("go-resolved-dependencies.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have 15 dependencies (excluding main module and local replacements)
	if len(deps) != 15 {
		t.Fatalf("expected 15 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Verify some key packages
	expected := map[string]struct {
		version string
		direct  bool
		scope   core.Scope
	}{
		"cloud.google.com/go":              {"v0.36.0", false, core.Runtime},
		"github.com/BurntSushi/toml":       {"v0.3.1", false, core.Runtime},
		"github.com/Masterminds/semver":    {"v1.5.0", false, core.Runtime},
		"github.com/Masterminds/semver/v3": {"v3.0.3", false, core.Runtime},
		"github.com/stretchr/testify":      {"v1.7.0", true, core.Test},
		"golang.org/x/net":                 {"v1.2.3", true, core.Runtime},
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
		if dep.Direct != exp.direct {
			t.Errorf("%s direct = %v, want %v", name, dep.Direct, exp.direct)
		}
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
		}
	}

	// Verify main module and local replacement are excluded
	if _, ok := depMap["main"]; ok {
		t.Error("main module should be excluded")
	}
	if _, ok := depMap["bad/thing"]; ok {
		t.Error("local replacement should be excluded")
	}
}

func TestGbManifest(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/gb_manifest")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gbManifestParser{}
	deps, err := parser.Parse("vendor/manifest", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.Name != "github.com/gorilla/mux" {
		t.Errorf("name = %q, want %q", dep.Name, "github.com/gorilla/mux")
	}
	if dep.Version != "9fa818a44c2bf1396a17f9d5a3c0f6dd39d2ff8e" {
		t.Errorf("version = %q, want %q", dep.Version, "9fa818a44c2bf1396a17f9d5a3c0f6dd39d2ff8e")
	}
	if dep.Scope != core.Runtime {
		t.Errorf("scope = %v, want Runtime", dep.Scope)
	}
}

func TestGodepsText(t *testing.T) {
	content, err := os.ReadFile("../../testdata/golang/Godeps")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &godepsTextParser{}
	deps, err := parser.Parse("Godeps", content)
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

	// All 5 packages with expected versions
	expected := map[string]string{
		"github.com/nu7hatch/gotrail":         "v0.0.2",
		"github.com/replicon/fast-archiver":   "v1.02",
		"github.com/garyburd/redigo/redis":    "a6a0a737c00caf4d4c2bb589941ace0d688168bb",
		"launchpad.net/gocheck":               "r2013.03.03",
		"code.google.com/p/go.example/hello/...": "ae081cd1d6cc",
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
		if dep.Scope != core.Runtime {
			t.Errorf("%s scope = %v, want Runtime", name, dep.Scope)
		}
		if !dep.Direct {
			t.Errorf("%s should be direct dependency", name)
		}
	}
}
