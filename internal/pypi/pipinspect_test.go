package pypi

import (
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPipInspectDeplock(t *testing.T) {
	depMap := parseFixture(t, "../../testdata/pypi/pip-inspect.deplock", "pip-inspect.deplock", &pipInspectParser{}, 6)

	if _, ok := depMap[""]; ok {
		t.Error("entry without a name should be skipped")
	}

	tests := []struct {
		name      string
		version   string
		direct    bool
		integrity string
		source    core.Source
	}{
		{name: "requests", version: "2.32.3", direct: true},
		{name: "urllib3", version: "2.2.3"},
		{name: "certifi", version: "2024.8.30"},
		{
			name: "attrs", version: "24.2.0", direct: true,
			source: core.Source{Kind: core.SourceGit, Value: "https://github.com/python-attrs/attrs.git", Ref: "a9a5ad6f2c4f8f1b6a1e3c8d2f4b5a6c7d8e9f01"},
		},
		{
			name: "six", version: "1.16.0", direct: true,
			integrity: "sha256-8abb2f1d86890a2dfb989f9a77cfcfd3e47c2a354b01111771326f8aa26e0254",
			source:    core.Source{Kind: core.SourceURL, Value: "https://files.pythonhosted.org/packages/d9/5a/e7c31adbe875f2abbb91bd84cf2dc52d792b5a01506781dbcf25c91daf11/six-1.16.0-py2.py3-none-any.whl"},
		},
		{
			name: "example-app", version: "0.1.0", direct: true,
			source: core.Source{Kind: core.SourcePath, Value: "/src/example-app"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep, ok := depMap[tt.name]
			if !ok {
				t.Fatalf("missing %s", tt.name)
			}
			if dep.Version != tt.version || dep.Direct != tt.direct || dep.Scope != core.Runtime {
				t.Errorf("got version %q direct %v scope %q", dep.Version, dep.Direct, dep.Scope)
			}
			if dep.Integrity != tt.integrity {
				t.Errorf("integrity = %q, want %q", dep.Integrity, tt.integrity)
			}
			if dep.Source != tt.source {
				t.Errorf("source = %+v, want %+v", dep.Source, tt.source)
			}
		})
	}
}

func TestPipInspectLegacyHash(t *testing.T) {
	content := []byte(`{"installed": [{"metadata": {"name": "six", "version": "1.16.0"},
		"direct_url": {"url": "https://example.com/six.whl", "archive_info": {"hash": "sha256=abc123"}}}]}`)
	res, err := (&pipInspectParser{}).Parse("pip-inspect.deplock", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Dependencies) != 1 || res.Dependencies[0].Integrity != "sha256-abc123" {
		t.Errorf("dependencies = %+v", res.Dependencies)
	}
}

func TestPipInspectInvalidJSON(t *testing.T) {
	if _, err := (&pipInspectParser{}).Parse("pip-inspect.deplock", []byte("{")); err == nil {
		t.Error("expected a parse error")
	}
}

func TestRequirementsDirInMatch(t *testing.T) {
	for name, want := range map[string]bool{
		"requirements/test.in":             true,
		"requirements/ci/lint.in":          true,
		"services/api/requirements/dev.in": true,
		"requirements/test.txt":            false,
		"requirements.in/dev.txt":          false,
		"dev.in":                           false,
		"myrequirements/dev.in":            false,
	} {
		if got := requirementsDirInMatch(name); got != want {
			t.Errorf("requirementsDirInMatch(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestNamedPylockMatch(t *testing.T) {
	for name, want := range map[string]bool{
		"pylock.dev.toml":        true,
		"sub/dir/pylock.ci.toml": true,
		"pylock.toml":            false,
		"pylock..toml":           false,
		"pylock.a.b.toml":        false,
		"pylock.dev.toml.bak":    false,
		"mypylock.dev.toml":      false,
	} {
		if got := namedPylockMatch(name); got != want {
			t.Errorf("namedPylockMatch(%q) = %v, want %v", name, got, want)
		}
	}
}
