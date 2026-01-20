package gem

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestGemfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/Gemfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileParser{}
	deps, err := parser.Parse("Gemfile", content)
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

	// All packages with exact versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"oj":            {"", core.Runtime},
		"rails":         {"4.2.0", core.Runtime},
		"leveldb-ruby":  {"0.15", core.Runtime},
		"nokogiri":      {"~> 1.6", core.Runtime},
		"rack":          {">= 2.0", core.Runtime},
		"json":          {"< 3.0", core.Runtime},
		"spring":        {"", core.Development},
		"thin":          {"", core.Development},
		"puma":          {"", core.Runtime}, // production group maps to runtime
		"rails_12factor": {"", core.Runtime},
		"bugsnag":       {"", core.Runtime},
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
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
		}
	}
}

func TestGemfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/Gemfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
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

	// All 4 packages from GEM specs with exact versions
	expected := map[string]struct {
		version string
		direct  bool
	}{
		"CFPropertyList": {"2.3.1", false},
		"actionmailer":   {"4.2.3", false},
		"googleauth":     {"0.4.1", false},
		"hashie":         {"3.4.2", false},
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
	}

	// Note: DEPENDENCIES section lists direct deps but without versions
	// Parser extracts from GEM specs which has resolved versions
}

func TestGemfileLockWithChecksums(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/Gemfile-with-checksums.lock")
	if err != nil {
		t.Skipf("fixture not found: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
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

	// All 4 packages from GEM specs with versions and integrities from CHECKSUMS
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"builder":         {"3.3.0", "sha256-497918d2f9dca528fdca4b88d84e4ef4387256d984b8154e9d5d3fe5a9c8835f"},
		"concurrent-ruby": {"1.3.6", "sha256-6b56837e1e7e5292f9864f34b69c5a2cbc75c0cf5338f1ce9903d10fa762d5ab"},
		"rack":            {"3.1.12", "sha256-40fcf876bb900016613cc330a7c331159c5bdc6f8bb60efdd9c4c0ba80e2ea0f"},
		"rake":            {"13.2.1", "sha256-46cb38dae65d7d74b6020a4ac9d48afed8eb8149c040eccf0523bec91907059d"},
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

func TestGemspec(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/devise.gemspec")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemspecParser{}
	deps, err := parser.Parse("devise.gemspec", content)
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

	// All packages with exact versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"warden":      {"~> 1.2.3", core.Runtime},
		"orm_adapter": {"~> 0.1", core.Development},
		"bcrypt":      {"~> 3.0", core.Runtime},
		"thread_safe": {"~> 0.1", core.Runtime},
		"railties":    {">= 3.2.6", core.Runtime}, // parser only captures first constraint
		"responders":  {"", core.Runtime},
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
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
		}
	}
}

func TestGemsRb(t *testing.T) {
	// gems.rb is an alternative name for Gemfile with identical format
	content, err := os.ReadFile("../../testdata/gem/gems.rb")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileParser{}
	deps, err := parser.Parse("gems.rb", content)
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

	// All 11 packages with exact versions and scopes (same as Gemfile)
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"oj":             {"", core.Runtime},
		"rails":          {"4.2.0", core.Runtime},
		"leveldb-ruby":   {"0.15", core.Runtime},
		"nokogiri":       {"~> 1.6", core.Runtime},
		"rack":           {">= 2.0", core.Runtime},
		"json":           {"< 3.0", core.Runtime},
		"spring":         {"", core.Development},
		"thin":           {"", core.Development},
		"puma":           {"", core.Runtime},
		"rails_12factor": {"", core.Runtime},
		"bugsnag":        {"", core.Runtime},
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
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", name, dep.Scope, exp.scope)
		}
	}
}

func TestGemfileLockLineEndings(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/GemfileLineEndings.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Single package
	if dep, ok := depMap["rails"]; !ok {
		t.Error("expected rails dependency")
	} else if dep.Version != "5.2.3" {
		t.Errorf("rails version = %q, want %q", dep.Version, "5.2.3")
	}
}

func TestGemfileLockWithBundler(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/GemfileWithBundler.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
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

	// All 4 packages from GEM specs (same as Gemfile.lock)
	expected := map[string]string{
		"CFPropertyList": "2.3.1",
		"actionmailer":   "4.2.3",
		"googleauth":     "0.4.1",
		"hashie":         "3.4.2",
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

func TestMastodonGemfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/mastodon/Gemfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Real-world mastodon lockfile has ~350 packages
	if len(deps) < 300 {
		t.Fatalf("expected at least 300 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with exact versions
	expected := map[string]string{
		"actioncable":              "8.0.3",
		"actionmailer":             "8.0.3",
		"actionpack":               "8.0.3",
		"activemodel":              "8.0.3",
		"activerecord":             "8.0.3",
		"activesupport":            "8.0.3",
		"addressable":              "2.8.8",
		"nokogiri":                 "1.18.10",
		"rack":                     "3.2.4",
		"rails":                    "8.0.3",
		"webpush":                  "1.1.0", // GIT source
		"active_model_serializers": "0.10.16",
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

func TestGemfileLockWithPlatforms(t *testing.T) {
	content, err := os.ReadFile("../../testdata/gem/GemfileWithPlatforms.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gemfileLockParser{}
	deps, err := parser.Parse("Gemfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should deduplicate platform variants: 7 unique packages
	if len(deps) != 7 {
		t.Fatalf("expected 7 dependencies (deduplicated), got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Versions should have platform suffixes stripped
	expected := map[string]string{
		"commonmarker":    "2.4.0",
		"nokogiri":        "1.18.0",
		"sassc":           "2.4.0",
		"google-protobuf": "4.30.0",
		"ffi":             "1.17.0",
		"grpc":            "1.70.0",
		"prerelease-gem":  "1.0.0-beta", // pre-release suffix should be preserved
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
