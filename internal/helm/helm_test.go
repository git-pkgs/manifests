package helm

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestChart(t *testing.T) {
	content, err := os.ReadFile("../../testdata/helm/Chart.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := (&chartParser{}).Parse("Chart.yaml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Name != "example-chart" {
		t.Errorf("Name = %q, want %q", result.Name, "example-chart")
	}
	if result.Version != "1.2.3" {
		t.Errorf("Version = %q, want %q", result.Version, "1.2.3")
	}

	assertHelmDependencies(t, result.Dependencies, map[string]helmDependencyExpectation{
		"postgresql":     {"~12.1.9", "https://charts.bitnami.com/bitnami", core.Source{}},
		"redis":          {"^17.3.0", "oci://registry-1.docker.io/bitnamicharts", core.Source{}},
		"metrics-server": {">=3.8.0 <4.0.0", "@internal", core.Source{}},
		"common":         {"1.x.x", "alias:partner", core.Source{}},
		"local-chart":    {"0.1.0", "", core.Source{Kind: core.SourcePath, Value: "../local-chart"}},
		"plugin-chart":   {"2.0.0", "s3://company-charts", core.Source{}},
	})
}

func TestChartWithoutDependencies(t *testing.T) {
	content, err := os.ReadFile("../../testdata/helm/minimal/Chart.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := (&chartParser{}).Parse("Chart.yaml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Name != "minimal" || result.Version != "0.1.0" {
		t.Errorf("package identity = %q %q, want minimal 0.1.0", result.Name, result.Version)
	}
	if len(result.Dependencies) != 0 {
		t.Errorf("Dependencies = %+v, want none", result.Dependencies)
	}
}

func TestChartLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/helm/Chart.lock")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	result, err := (&chartLockParser{}).Parse("Chart.lock", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Digest != "sha256:8ca45f73ae3f6170a09b64a967006e98e13cd91eb51e5ab0599bb87296c7df0a" {
		t.Errorf("Digest = %q", result.Digest)
	}

	assertHelmDependencies(t, result.Dependencies, map[string]helmDependencyExpectation{
		"postgresql":     {"12.1.15", "https://charts.bitnami.com/bitnami", core.Source{}},
		"redis":          {"17.3.7", "oci://registry-1.docker.io/bitnamicharts", core.Source{}},
		"metrics-server": {"3.12.2", "@internal", core.Source{}},
		"common":         {"1.17.1", "alias:partner", core.Source{}},
		"local-chart":    {"0.1.0", "", core.Source{Kind: core.SourcePath, Value: "../local-chart"}},
		"plugin-chart":   {"2.0.0", "s3://company-charts", core.Source{}},
	})
	for _, dependency := range result.Dependencies {
		if dependency.Integrity != "" {
			t.Errorf("%s Integrity = %q, want empty", dependency.Name, dependency.Integrity)
		}
	}

	changedGenerated := strings.Replace(
		string(content),
		`generated: "2021-05-02T15:07:22.1099921+02:00"`,
		`generated: "2026-08-16T09:00:00Z"`,
		1,
	)
	if changedGenerated == string(content) {
		t.Fatal("generated timestamp was not replaced")
	}
	changedResult, err := (&chartLockParser{}).Parse("Chart.lock", []byte(changedGenerated))
	if err != nil {
		t.Fatalf("Parse with changed generated timestamp: %v", err)
	}
	if !reflect.DeepEqual(changedResult, result) {
		t.Errorf("generated timestamp changed result:\n got %+v\nwant %+v", changedResult, result)
	}
}

func TestChartLockMissingOptionalFields(t *testing.T) {
	content := []byte("dependencies:\n- name: bundled\n  version: 1.2.3\n")
	result, err := (&chartLockParser{}).Parse("Chart.lock", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if result.Digest != "" {
		t.Errorf("Digest = %q, want empty", result.Digest)
	}
	if len(result.Dependencies) != 1 {
		t.Fatalf("Dependencies has %d entries, want 1", len(result.Dependencies))
	}
	dependency := result.Dependencies[0]
	if dependency.Name != "bundled" || dependency.Version != "1.2.3" || dependency.RegistryURL != "" {
		t.Errorf("unexpected dependency: %+v", dependency)
	}
}

type helmDependencyExpectation struct {
	version     string
	registryURL string
	source      core.Source
}

func assertHelmDependencies(t *testing.T, got []core.Dependency, want map[string]helmDependencyExpectation) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Dependencies has %d entries, want %d", len(got), len(want))
	}
	seen := make(map[string]bool, len(got))
	for _, dependency := range got {
		expected, ok := want[dependency.Name]
		if !ok {
			t.Errorf("unexpected dependency: %+v", dependency)
			continue
		}
		seen[dependency.Name] = true
		if dependency.Version != expected.version {
			t.Errorf("%s Version = %q, want %q", dependency.Name, dependency.Version, expected.version)
		}
		if dependency.RegistryURL != expected.registryURL {
			t.Errorf("%s RegistryURL = %q, want %q", dependency.Name, dependency.RegistryURL, expected.registryURL)
		}
		if dependency.Source != expected.source {
			t.Errorf("%s Source = %+v, want %+v", dependency.Name, dependency.Source, expected.source)
		}
		if dependency.Scope != core.Runtime || !dependency.Direct {
			t.Errorf("unexpected dependency metadata: %+v", dependency)
		}
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("missing dependency %q", name)
		}
	}
}

func TestMalformedYAML(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		parser   core.Parser
	}{
		{name: "chart", filename: "Chart.yaml", parser: &chartParser{}},
		{name: "lock", filename: "Chart.lock", parser: &chartLockParser{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.parser.Parse(test.filename, []byte("dependencies: ["))
			if err == nil {
				t.Fatal("Parse returned nil error")
			}
			var parseError *core.ParseError
			if !errors.As(err, &parseError) {
				t.Errorf("error = %T, want *core.ParseError", err)
			}
		})
	}
}
