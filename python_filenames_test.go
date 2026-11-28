package manifests

import (
	"os"
	"reflect"
	"testing"
)

func TestPythonRequirementsInFiles(t *testing.T) {
	content, err := os.ReadFile("testdata/pypi/requirements/test.in")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"requirements.in", "requirements-dev.in", "requirements/test.in",
		"requirements/ci/lint.in", "services/api/requirements/test.in",
	} {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(name, content)
			if err != nil {
				t.Fatal(err)
			}
			if got.Ecosystem != "pypi" || got.Kind != Manifest {
				t.Fatalf("identified as %s %s", got.Ecosystem, got.Kind)
			}
			want := map[string]string{"pexpect": "", "pytest": ">=7", "pytest-xdist": "", "exceptiongroup": ""}
			if len(got.Dependencies) != len(want) || len(got.Declarations) != len(want) {
				t.Fatalf("dependencies = %+v", got.Dependencies)
			}
			for _, dep := range got.Dependencies {
				version, ok := want[dep.Name]
				if !ok || dep.Version != version || !dep.Direct || dep.Scope != Runtime {
					t.Errorf("unexpected dependency %+v", dep)
				}
			}
		})
	}
}

func TestPythonNamedPylockFiles(t *testing.T) {
	content, err := os.ReadFile("testdata/pypi/pylock.dev.toml")
	if err != nil {
		t.Fatal(err)
	}
	want, err := Parse("pylock.toml", content)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.Dependencies) != 2 {
		t.Fatalf("fixture dependencies = %+v", want.Dependencies)
	}
	for _, name := range []string{"pylock.dev.toml", "pylock.ci.toml", "nested/dir/pylock.dev.toml"} {
		t.Run(name, func(t *testing.T) {
			got, err := Parse(name, content)
			if err != nil {
				t.Fatal(err)
			}
			if got.Ecosystem != "pypi" || got.Kind != Lockfile {
				t.Fatalf("identified as %s %s", got.Ecosystem, got.Kind)
			}
			if !reflect.DeepEqual(got.Dependencies, want.Dependencies) {
				t.Errorf("dependencies differ from pylock.toml")
			}
		})
	}
}

func TestPythonPipInspectDeplock(t *testing.T) {
	content, err := os.ReadFile("testdata/pypi/pip-inspect.deplock")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse("pip-inspect.deplock", content)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ecosystem != "pypi" || got.Kind != Lockfile || len(got.Dependencies) != 6 {
		t.Fatalf("got %s %s with %d dependencies", got.Ecosystem, got.Kind, len(got.Dependencies))
	}
	first := got.Dependencies[0]
	if first.Name != "requests" || first.PURL != "pkg:pypi/requests@2.32.3" || !first.Direct {
		t.Errorf("first dependency = %+v", first)
	}
}

func TestPythonSimilarNamesNotMatched(t *testing.T) {
	for _, name := range []string{
		"dev.in", "requirements.in.bak", "pylock.a.b.toml", "pylock..toml",
		"pylock.dev.toml.bak", "pip-inspect.json", "my-pip-inspect.deplock",
	} {
		if _, _, ok := Identify(name); ok {
			t.Errorf("%s should not be identified", name)
		}
	}
}
