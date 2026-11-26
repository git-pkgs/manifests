package manifests

import (
	"reflect"
	"strings"
	"testing"
)

func TestPyprojectBuildSystemRequires(t *testing.T) {
	result := parsePython(t, "pyproject.toml", `[build-system]
requires = ["setuptools>=68"]
build-backend = "setuptools.build_meta"

[project]
name = "example"
version = "0.1.0"
`)
	if result.Name != "example" || result.Version != "0.1.0" {
		t.Errorf("identity = %s %s", result.Name, result.Version)
	}
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"setuptools", ">=68", Build, "pkg:pypi/setuptools", nil, ""},
	})
	want := []Declaration{{
		Name:        "setuptools",
		Version:     ">=68",
		Scope:       Build,
		Direct:      true,
		PURL:        "pkg:pypi/setuptools",
		Ecosystem:   "pypi",
		Location:    "build-system/requires/setuptools",
		Requirement: "setuptools>=68",
	}}
	if !reflect.DeepEqual(result.Declarations, want) {
		t.Errorf("declarations =\n%+v\nwant\n%+v", result.Declarations, want)
	}
	for _, dep := range result.Dependencies {
		if strings.Contains(dep.Name, "build_meta") {
			t.Errorf("build-backend recorded as dependency %+v", dep)
		}
	}
}

func TestPyprojectBuildSystemMultipleRequires(t *testing.T) {
	result := parsePython(t, "pyproject.toml", `[build-system]
requires = [
  "setuptools>=68",
  "wheel",
  "setuptools-scm[toml]>=8; python_version >= '3.8'",
  "Cython (>=3.0)",
  "setuptools<70; sys_platform == 'win32'",
]
build-backend = "setuptools.build_meta"
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"setuptools", ">=68", Build, "pkg:pypi/setuptools", nil, ""},
		{"wheel", "", Build, "pkg:pypi/wheel", nil, ""},
		{"setuptools-scm", ">=8", Build, "pkg:pypi/setuptools-scm", []string{"toml"}, "python_version >= '3.8'"},
		{"Cython", ">=3.0", Build, "pkg:pypi/cython", nil, ""},
		{"setuptools", "<70", Build, "pkg:pypi/setuptools", nil, "sys_platform == 'win32'"},
	})
	wantLocations := []string{
		"build-system/requires/setuptools",
		"build-system/requires/wheel",
		"build-system/requires/setuptools-scm",
		"build-system/requires/cython",
		"build-system/requires/setuptools/2",
	}
	if len(result.Declarations) != len(wantLocations) {
		t.Fatalf("declarations = %+v", result.Declarations)
	}
	for i, declaration := range result.Declarations {
		if declaration.Location != wantLocations[i] || declaration.Scope != Build {
			t.Errorf("declaration %d = %+v, want build scope at %s", i, declaration, wantLocations[i])
		}
	}
}

func TestPyprojectBuildSystemEmptyRequires(t *testing.T) {
	tests := map[string]string{
		"empty list":      "[build-system]\nrequires = []\nbuild-backend = \"flit_core.buildapi\"\n",
		"backend only":    "[build-system]\nbuild-backend = \"pdm.backend\"\n",
		"no build-system": "[project]\nname = \"example\"\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			result := parsePython(t, "pyproject.toml", content)
			if len(result.Dependencies) != 0 || len(result.Declarations) != 0 {
				t.Errorf("dependencies = %+v, declarations = %+v, want none", result.Dependencies, result.Declarations)
			}
		})
	}
}

func TestPyprojectBuildSystemWithProjectDependencies(t *testing.T) {
	result := parsePython(t, "pyproject.toml", `[build-system]
requires = ["hatchling>=1.18"]
build-backend = "hatchling.build"

[project]
name = "example"
dependencies = ["requests>=2"]

[project.optional-dependencies]
test = ["pytest>=8"]
`)
	assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
		{"requests", ">=2", Runtime, "pkg:pypi/requests", nil, ""},
		{"pytest", ">=8", Test, "pkg:pypi/pytest", nil, ""},
		{"hatchling", ">=1.18", Build, "pkg:pypi/hatchling", nil, ""},
	})
	var locations []string
	for _, declaration := range result.Declarations {
		locations = append(locations, declaration.Location)
	}
	want := []string{
		"project/dependencies/requests",
		"project/optional-dependencies/test/pytest",
		"build-system/requires/hatchling",
	}
	if !reflect.DeepEqual(locations, want) {
		t.Errorf("locations = %v, want %v", locations, want)
	}
}

func TestPyprojectBuildSystemMalformedRequires(t *testing.T) {
	tests := map[string]string{
		"string":         "[build-system]\nrequires = \"setuptools\"\n\n[project]\ndependencies = [\"requests\"]\n",
		"table":          "[build-system.requires]\nsetuptools = \"68\"\n\n[project]\ndependencies = [\"requests\"]\n",
		"mixed elements": "[build-system]\nrequires = [1, \"./local\", true]\n\n[project]\ndependencies = [\"requests\"]\n",
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			result := parsePython(t, "pyproject.toml", content)
			assertPythonRequirements(t, result.Dependencies, []pythonRequirement{
				{"requests", "", Runtime, "pkg:pypi/requests", nil, ""},
			})
		})
	}
}
