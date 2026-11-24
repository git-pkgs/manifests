package manifests

import (
	"reflect"
	"testing"
)

// pythonRequirement is the part of a dependency these tests check.
type pythonRequirement struct {
	Name    string
	Version string
	Scope   Scope
	PURL    string
	Extras  []string
	Marker  string
}

func pythonRequirements(deps []Dependency) []pythonRequirement {
	got := make([]pythonRequirement, 0, len(deps))
	for _, dep := range deps {
		got = append(got, pythonRequirement{dep.Name, dep.Version, dep.Scope, dep.PURL, dep.Extras, dep.Marker})
	}
	return got
}

func parsePython(t *testing.T, filename, content string) *ParseResult {
	t.Helper()
	result, err := Parse(filename, []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func assertPythonRequirements(t *testing.T, deps []Dependency, want []pythonRequirement) {
	t.Helper()
	if got := pythonRequirements(deps); !reflect.DeepEqual(got, want) {
		t.Errorf("dependencies =\n%+v\nwant\n%+v", got, want)
	}
}

func TestRequirementsTxtExtrasAndMarkers(t *testing.T) {
	result := parsePython(t, "requirements.txt", `requests[socks]>=2
pywin32>=300; sys_platform == 'win32'
urllib3[secure, socks] == 1.26.0 ; python_version < "3.12"  # pinned
plain==1.0
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"requests", ">=2", Runtime, "pkg:pypi/requests", []string{"socks"}, ""},
		{"pywin32", ">=300", Runtime, "pkg:pypi/pywin32", nil, "sys_platform == 'win32'"},
		{"urllib3", "== 1.26.0", Runtime, "pkg:pypi/urllib3", []string{"secure", "socks"}, `python_version < "3.12"`},
		{"plain", "==1.0", Runtime, "pkg:pypi/plain", nil, ""},
	})

	want := []Declaration{
		{Name: "requests", Version: ">=2", Scope: Runtime, Direct: true, PURL: "pkg:pypi/requests", Ecosystem: "pypi", Location: "requirements/requests", Extras: []string{"socks"}, Requirement: "requests[socks]>=2"},
		{Name: "pywin32", Version: ">=300", Scope: Runtime, Direct: true, PURL: "pkg:pypi/pywin32", Ecosystem: "pypi", Location: "requirements/pywin32", Marker: "sys_platform == 'win32'", Requirement: "pywin32>=300; sys_platform == 'win32'"},
		{Name: "urllib3", Version: "== 1.26.0", Scope: Runtime, Direct: true, PURL: "pkg:pypi/urllib3", Ecosystem: "pypi", Location: "requirements/urllib3", Extras: []string{"secure", "socks"}, Marker: `python_version < "3.12"`, Requirement: `urllib3[secure, socks] == 1.26.0 ; python_version < "3.12"`},
		{Name: "plain", Version: "==1.0", Scope: Runtime, Direct: true, PURL: "pkg:pypi/plain", Ecosystem: "pypi", Location: "requirements/plain", Requirement: "plain==1.0"},
	}
	if !reflect.DeepEqual(result.Declarations, want) {
		t.Errorf("declarations =\n%+v\nwant\n%+v", result.Declarations, want)
	}
}

func TestRequirementsTxtRepeatedUnderDifferentMarkers(t *testing.T) {
	result := parsePython(t, "requirements.txt", `numpy>=1.26; python_version >= '3.12'
numpy>=1.21,<1.26; python_version < '3.12'
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"numpy", ">=1.26", Runtime, "pkg:pypi/numpy", nil, "python_version >= '3.12'"},
		{"numpy", ">=1.21,<1.26", Runtime, "pkg:pypi/numpy", nil, "python_version < '3.12'"},
	})
	if len(result.Declarations) != 2 {
		t.Fatalf("declarations = %+v", result.Declarations)
	}
	first, second := result.Declarations[0], result.Declarations[1]
	if first.Location != "requirements/numpy" || first.Marker != "python_version >= '3.12'" {
		t.Errorf("first declaration = %+v", first)
	}
	if second.Location != "requirements/numpy/2" || second.Marker != "python_version < '3.12'" {
		t.Errorf("second declaration = %+v", second)
	}
}

func TestRequirementsTxtLineHandling(t *testing.T) {
	result := parsePython(t, "requirements.txt", `certifi==2024.2.2 \
    --hash=sha256:0123
idna==3.6 ; python_version >= '3.8' --hash=sha256:4567
pip @ https://example.test/pip-24.0-py3-none-any.whl#sha256=89ab ; python_version >= '3.8'
git+https://github.com/pygame/pygame@2.1.2#egg=pygame
-e ./local
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"certifi", "==2024.2.2", Runtime, "pkg:pypi/certifi", nil, ""},
		{"idna", "==3.6", Runtime, "pkg:pypi/idna", nil, "python_version >= '3.8'"},
		{"pip", "", Runtime, "pkg:pypi/pip", nil, "python_version >= '3.8'"},
	})
	if got := result.Declarations[2].Requirement; got != "pip @ https://example.test/pip-24.0-py3-none-any.whl#sha256=89ab ; python_version >= '3.8'" {
		t.Errorf("direct reference requirement = %q", got)
	}
}

func TestPyprojectExtrasAndMarkers(t *testing.T) {
	result := parsePython(t, "pyproject.toml", `[project]
name = "app"
dependencies = [
  "requests[socks]>=2; python_version < '3.12'",
  "requests>=2.32; python_version >= '3.12'",
  "importlib-metadata[perf] (>=6.0)",
]

[project.optional-dependencies]
test = ["pytest[testing]>=8 ; sys_platform != 'win32'"]
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"requests", ">=2", Runtime, "pkg:pypi/requests", []string{"socks"}, "python_version < '3.12'"},
		{"requests", ">=2.32", Runtime, "pkg:pypi/requests", nil, "python_version >= '3.12'"},
		{"importlib-metadata", ">=6.0", Runtime, "pkg:pypi/importlib-metadata", []string{"perf"}, ""},
		{"pytest", ">=8", Test, "pkg:pypi/pytest", []string{"testing"}, "sys_platform != 'win32'"},
	})

	wantLocations := []string{
		"project/dependencies/requests",
		"project/dependencies/requests/2",
		"project/dependencies/importlib-metadata",
		"project/optional-dependencies/test/pytest",
	}
	if len(result.Declarations) != len(wantLocations) {
		t.Fatalf("declarations = %+v", result.Declarations)
	}
	for i, declaration := range result.Declarations {
		dep := result.Dependencies[i]
		if declaration.Location != wantLocations[i] || declaration.Marker != dep.Marker || !reflect.DeepEqual(declaration.Extras, dep.Extras) {
			t.Errorf("declaration %d = %+v, want location %s with the dependency's extras and marker", i, declaration, wantLocations[i])
		}
	}
	if got := result.Declarations[3].Requirement; got != "pytest[testing]>=8 ; sys_platform != 'win32'" {
		t.Errorf("requirement = %q", got)
	}
}

func TestSetupCfgExtrasAndMarkers(t *testing.T) {
	result := parsePython(t, "setup.cfg", `[metadata]
name = app

[options]
install_requires =
    requests[socks]>=2; python_version < '3.12'
    six

[options.extras_require]
test =
    pytest>=8; sys_platform != 'win32'
docs = sphinx[standalone]
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"requests", ">=2", Runtime, "pkg:pypi/requests", []string{"socks"}, "python_version < '3.12'"},
		{"six", "", Runtime, "pkg:pypi/six", nil, ""},
		{"sphinx", "", Optional, "pkg:pypi/sphinx", []string{"standalone"}, ""},
		{"pytest", ">=8", Test, "pkg:pypi/pytest", nil, "sys_platform != 'win32'"},
	})
}

func TestSetupPyExtrasAndMarkers(t *testing.T) {
	result := parsePython(t, "setup.py", `from setuptools import setup

setup(
    name="app",
    install_requires=[
        'requests[socks]>=2; python_version < "3.12"',  # extras with a "]"
        "pywin32>=300; sys_platform == 'win32'",
        "six",
    ],
    extras_require={
        "test": ["pytest[testing]>=8; python_version >= '3.8'"],
        'docs': ['sphinx'],
    },
)
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"requests", ">=2", Runtime, "pkg:pypi/requests", []string{"socks"}, `python_version < "3.12"`},
		{"pywin32", ">=300", Runtime, "pkg:pypi/pywin32", nil, "sys_platform == 'win32'"},
		{"six", "", Runtime, "pkg:pypi/six", nil, ""},
		{"pytest", ">=8", Test, "pkg:pypi/pytest", []string{"testing"}, "python_version >= '3.8'"},
		{"sphinx", "", Optional, "pkg:pypi/sphinx", nil, ""},
	})
}

func TestSetupPyInstallRequiresVariable(t *testing.T) {
	result := parsePython(t, "setup.py", `REQUIRES = ["click"]
install_requires = ["requests[socks]>=2"]
setup(name="app", install_requires=install_requires)
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"requests", ">=2", Runtime, "pkg:pypi/requests", []string{"socks"}, ""},
	})
}
