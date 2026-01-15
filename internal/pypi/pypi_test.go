package pypi

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestRequirementsTxt(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/requirements.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.txt", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 10 {
		t.Fatalf("expected 10 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 10 packages with versions
	expected := map[string]string{
		"Flask":        "== 0.8",
		"zope.component": "==4.2.2",
		"scikit-learn": "==0.16.1",
		"Beaker":       ">=1.6.5",
		"certifi":      "==0.0.8",
		"chardet":      "==1.0.1",
		"distribute":   "==0.6.24",
		"gunicorn":     "==0.14.2",
		"requests":     "==0.11.1",
		"Django":       "== 2.0beta1",
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

	// Verify comment was filtered out
	if _, ok := depMap["Jinja2"]; ok {
		t.Error("commented Jinja2 should not be included")
	}
}

func TestPipfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/Pipfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pipfileParser{}
	deps, err := parser.Parse("Pipfile", content)
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

	// All 7 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"pinax", "*", core.Runtime},
		{"a-local-dep", "*", core.Runtime},
		{"urllib3", "*", core.Runtime},
		{"requests", "*", core.Runtime},
		{"Django", ">1.10", core.Runtime},
		{"another-local-dep", "*", core.Development},
		{"nose", "*", core.Development},
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
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}

func TestPipfileLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/Pipfile.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pipfileLockParser{}
	deps, err := parser.Parse("Pipfile.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 13 {
		t.Fatalf("expected 13 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions and integrities
	expected := []struct {
		name      string
		version   string
		integrity string
	}{
		{"asgiref", "3.9.1", "sha256-a5ab6582236218e5ef1648f242fd9f10626cfd4de8dc377db215d5d5098e3142"},
		{"certifi", "2025.7.14", "sha256-6b31f564a415d79ee77df69d757bb49a5bb53bd9f756cbbe24394ffd6fc1f4b2"},
		{"idna", "3.10", "sha256-12f65c9b470abda6dc35cf8e63cc574b1c52b11df2c86030af0ac09b01b13ea9"},
		{"django", "5.2.4", "sha256-60c35bd96201b10c6e7a78121bd0da51084733efa303cc19ead021ab179cef5e"},
		{"requests", "2.32.4", "sha256-27babd3cda2a6d50b30443204ee89830707d396671944c998b5975b031ac2b2c"},
		{"nose", "1.3.7", "sha256-9ff7c6cc443f8c51994b34a667bbcf45afd6d945be7477b52e97516fd17c53ac"},
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

	// Verify nose is dev scope
	if dep, ok := depMap["nose"]; ok {
		if dep.Scope != core.Development {
			t.Errorf("nose scope = %v, want Development", dep.Scope)
		}
	}
}

func TestPyprojectToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pyproject.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pyprojectParser{}
	deps, err := parser.Parse("pyproject.toml", content)
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

	// All 6 packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"Zope_interface", "6.3", core.Runtime},
		{"django", "^3.0.7", core.Runtime},
		{"pathlib2", "2.3.7.post1", core.Runtime},
		{"pytest", "^5.2", core.Development},
		{"sqlparse", "0.4.4", core.Test},
		{"wcwidth", "*", core.Development},
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
		if dep.Scope != exp.scope {
			t.Errorf("%s scope = %v, want %v", exp.name, dep.Scope, exp.scope)
		}
	}
}

func TestPoetryLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/poetry.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &poetryLockParser{}
	deps, err := parser.Parse("poetry.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 20 {
		t.Fatalf("expected 20 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 20 packages with versions
	expected := map[string]string{
		"asgiref":             "3.7.2",
		"atomicwrites":        "1.4.1",
		"attrs":               "24.2.0",
		"colorama":            "0.4.6",
		"django":              "3.2.25",
		"importlib-metadata":  "6.7.0",
		"more-itertools":      "9.1.0",
		"packaging":           "24.0",
		"pathlib2":            "2.3.7.post1",
		"pluggy":              "0.13.1",
		"py":                  "1.11.0",
		"pytest":              "5.4.3",
		"pytz":                "2025.2",
		"setuptools":          "68.0.0",
		"six":                 "1.17.0",
		"sqlparse":            "0.4.4",
		"typing-extensions":   "4.7.1",
		"wcwidth":             "0.2.13",
		"zipp":                "3.15.0",
		"zope-interface":      "6.3",
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

func TestRequirementsDevTxt(t *testing.T) {
	// Tests pip-compile output format (requirements-dev.txt)
	content, err := os.ReadFile("../../testdata/pypi/requirements-dev.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements-dev.txt", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 51 {
		t.Fatalf("expected 51 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"astroid":            "==2.9.0",
		"attrs":              "==21.4.0",
		"boto3":              "==1.20.26",
		"coverage":           "==6.2",
		"flake8":             "==4.0.1",
		"pytest":             "==6.2.5",
		"mypy":               "==0.812",
		"pylint":             "==2.12.2",
		"requests":           "==2.26.0",
		"wheel":              "==0.37.1",
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

func TestRequirementsFrozen(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/requirements.frozen")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.frozen", content)
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

	// All 3 packages with versions
	expected := map[string]string{
		"asgiref":  "==3.2.7",
		"Django":   "==3.0.6",
		"sqlparse": "==0.3.1",
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

func TestPdmLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pdm.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pdmLockParser{}
	deps, err := parser.Parse("pdm.lock", content)
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

	// All 3 packages with exact versions, scopes, and integrities
	// certifi - default group (runtime)
	if dep, ok := depMap["certifi"]; !ok {
		t.Error("expected certifi dependency")
	} else {
		if dep.Version != "2024.2.2" {
			t.Errorf("certifi version = %q, want %q", dep.Version, "2024.2.2")
		}
		if dep.Scope != core.Runtime {
			t.Errorf("certifi scope = %v, want Runtime", dep.Scope)
		}
		wantIntegrity := "sha256-dc383c07b76109f368f6106eee2b593b04a011ea4d55f652c6ca24a754d1cdd1"
		if dep.Integrity != wantIntegrity {
			t.Errorf("certifi integrity = %q, want %q", dep.Integrity, wantIntegrity)
		}
	}

	// requests - default group (runtime)
	if dep, ok := depMap["requests"]; !ok {
		t.Error("expected requests dependency")
	} else {
		if dep.Version != "2.31.0" {
			t.Errorf("requests version = %q, want %q", dep.Version, "2.31.0")
		}
		if dep.Scope != core.Runtime {
			t.Errorf("requests scope = %v, want Runtime", dep.Scope)
		}
		wantIntegrity := "sha256-58cd2187c01e70e6e26505bca751777aa9f2ee0b7f4300988b709f44e013003f"
		if dep.Integrity != wantIntegrity {
			t.Errorf("requests integrity = %q, want %q", dep.Integrity, wantIntegrity)
		}
	}

	// pytest - dev group (development)
	if dep, ok := depMap["pytest"]; !ok {
		t.Error("expected pytest dependency")
	} else {
		if dep.Version != "8.0.0" {
			t.Errorf("pytest version = %q, want %q", dep.Version, "8.0.0")
		}
		if dep.Scope != core.Development {
			t.Errorf("pytest scope = %v, want Development", dep.Scope)
		}
		wantIntegrity := "sha256-50fb9cbe836c3f20f0619a2e6a5e137e8e4d3e9c9e2e4e85c66a3d5a8d8e9a0f"
		if dep.Integrity != wantIntegrity {
			t.Errorf("pytest integrity = %q, want %q", dep.Integrity, wantIntegrity)
		}
	}
}

func TestUvLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/uv.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &uvLockParser{}
	deps, err := parser.Parse("uv.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 36 {
		t.Fatalf("expected 36 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with exact versions and integrities (from sdist hashes)
	expected := map[string]struct {
		version   string
		integrity string
	}{
		"alabaster":          {"0.7.16", "sha256-75a8b99c28a5dad50dd7f8ccdd447a121ddb3892da9e53d1ca5cca3106d58d65"},
		"babel":              {"2.16.0", "sha256-d1f3554ca26605fe173f3de0c65f750f5a42f924499bf134de6423582298e316"},
		"beautifulsoup4":     {"4.12.3", "sha256-74e3d1928edc070d21748185c46e3fb33490f22f52a3addee9aee0f4f7781051"},
		"certifi":            {"2024.8.30", "sha256-bec941d2aa8195e248a60b31ff9f0558284cf01a52591ceda73ea9afffd69fd9"},
		"charset-normalizer": {"3.4.0", "sha256-223217c3d4f82c3ac5e29032b3f1c2eb0fb591b72161f86d93f5719079dae93e"},
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

func TestParsePEP508(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"requests>=2.0", "requests", ">=2.0"},
		{"requests[security]>=2.0", "requests", ">=2.0"},
		{"Django>=3.0,<4.0", "Django", ">=3.0,<4.0"},
		{"pytest", "pytest", ""},
		{"black==22.3.0", "black", "==22.3.0"},
		{"numpy>=1.20; python_version>='3.8'", "numpy", ">=1.20"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotName, gotVer := parsePEP508(tt.input)
			if gotName != tt.wantName {
				t.Errorf("name = %q, want %q", gotName, tt.wantName)
			}
			if gotVer != tt.wantVersion {
				t.Errorf("version = %q, want %q", gotVer, tt.wantVersion)
			}
		})
	}
}

func TestPipCompileRequirementsIn(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pip-compile/requirements.in")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.in", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 10 {
		t.Fatalf("expected 10 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 10 packages (no versions in .in files)
	expectedNames := []string{
		"invoke", "black", "google-cloud-storage", "six", "progress",
		"questionary", "pyyaml", "semver", "Jinja2", "pip-tools",
	}

	for _, name := range expectedNames {
		if _, ok := depMap[name]; !ok {
			t.Errorf("expected %s dependency", name)
		}
	}
}

func TestPipCompileRequirementsTxt(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pip-compile/requirements.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.txt", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 38 {
		t.Fatalf("expected 38 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"black":                  "==21.9b0",
		"google-cloud-storage":  "==1.42.2",
		"invoke":                 "==1.6.0",
		"jinja2":                 "==3.0.1",
		"pip-tools":              "==6.2.0",
		"pyyaml":                 "==5.4.1",
		"questionary":            "==1.10.0",
		"semver":                 "==2.13.0",
		"six":                    "==1.16.0",
		"requests":               "==2.26.0",
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

func TestPipCompileFrozen(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pip-compile/requirements.frozen")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements.frozen", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	if deps[0].Name != "black" {
		t.Errorf("expected black, got %s", deps[0].Name)
	}
	if deps[0].Version != "==21.9b0" {
		t.Errorf("black version = %q, want %q", deps[0].Version, "==21.9b0")
	}
}

func TestPipCompileExtrasIn(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pip-compile/requirements-extras.in")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("requirements-extras.in", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 2 packages
	expected := map[string]string{
		"urllib3":            "==1.0.0",
		"django-dbfilestorage": "==1.0.0",
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

func TestPoetryProjectLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/poetry-project/poetry.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &poetryLockParser{}
	deps, err := parser.Parse("poetry.lock", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 77 {
		t.Fatalf("expected 77 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"anyio":              "4.12.0",
		"backports-tarfile": "1.2.0",
		"build":              "1.3.0",
		"certifi":            "2025.11.12",
		"coverage":           "7.12.0",
		"cryptography":       "46.0.3",
		"deepdiff":           "8.6.1",
		"dulwich":            "0.25.0",
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

func TestRequirementsTestTxt(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/requirements/test.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &requirementsTxtParser{}
	deps, err := parser.Parse("test.txt", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 15 {
		t.Fatalf("expected 15 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"attrs":        "==21.4.0",
		"execnet":      "==1.9.0",
		"iniconfig":    "==1.1.1",
		"packaging":    "==21.3",
		"pytest":       "==7.1.2",
		"pytest-xdist": "==2.5.0",
		"tomli":        "==2.0.1",
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

func TestPipDependencyGraph(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pip-dependency-graph.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pipDependencyGraphParser{}
	deps, err := parser.Parse("pip-dependency-graph.json", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 17 {
		t.Fatalf("expected 17 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample packages with versions
	expected := map[string]string{
		"aiohttp":         "3.9.5",
		"aiosignal":       "1.3.1",
		"black":           "23.12.0",
		"click":           "8.1.7",
		"frozenlist":      "1.4.1",
		"idna":            "3.7",
		"multidict":       "6.0.5",
		"mypy-extensions": "1.0.0",
		"packaging":       "24.0",
		"pathspec":        "0.12.1",
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

func TestPipResolvedDeps(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pip-resolved-dependencies.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pipResolvedDepsParser{}
	deps, err := parser.Parse("pip-resolved-dependencies.txt", content)
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

	expected := map[string]string{
		"asgiref":  "3.2.7",
		"Django":   "3.0.6",
		"sqlparse": "0.3.1",
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

func TestSetupPy(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/setup.py")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &setupPyParser{}
	deps, err := parser.Parse("setup.py", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) < 17 {
		t.Fatalf("expected at least 17 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of install_requires packages
	installRequires := map[string]string{
		"django-bootstrap3":          ">=6.2,<6.3",
		"lesscpy":                    "",
		"unicodecsv":                 "==0.14.1",
		"django-coffeescript":        ">=0.7,<0.8",
		"django-compressor":          ">=1.6,<1.7",
		"django-filter":              ">=0.11,<0.12",
		"django-representatives-votes": ">=0.0.13",
		"django-representatives":     ">=0.0.14",
		"django":                     ">=1.8,<1.9",
		"pytz":                       "==2015.7",
	}

	for name, wantVer := range installRequires {
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

func TestPylockToml(t *testing.T) {
	content, err := os.ReadFile("../../testdata/pypi/pylock.toml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pylockTomlParser{}
	deps, err := parser.Parse("pylock.toml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 20 {
		t.Fatalf("expected 20 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample packages with versions and integrities
	expected := []struct {
		name      string
		version   string
		integrity string
	}{
		{"blinker", "1.9.0", "sha256-ba0efaa9080b619ff2f3459d1d500c57bddea4a6b424b60a91141db6fd2f08bc"},
		{"certifi", "2025.7.14", "sha256-6b31f564a415d79ee77df69d757bb49a5bb53bd9f756cbbe24394ffd6fc1f4b2"},
		{"click", "8.2.1", "sha256-61a3265b914e850b85317d0b3109c7f8cd35a670f963866005d6ef1d5175a12b"},
		{"flask", "3.1.1", "sha256-07aae2bb5eaf77993ef57e357491839f5fd9f4dc281593a81a9e4d79a24f295c"},
		{"requests", "2.32.4", "sha256-27babd3cda2a6d50b30443204ee89830707d396671944c998b5975b031ac2b2c"},
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
