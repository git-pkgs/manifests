package manifests

import (
	"fmt"
	"strings"
	"testing"
)

func TestSwiftManifestRequirements(t *testing.T) {
	tests := []struct {
		name        string
		declaration string
		want        string
		branch      string
		ref         string
	}{
		{"exact", `exact: "1.2.3"`, "1.2.3", "", ""},
		{"legacy exact", `.exact("1.2.3")`, "1.2.3", "", ""},
		{"from", `from: "1.2.3"`, ">=1.2.3 <2.0.0", "", ""},
		{"from zero", `from: "0.2.3"`, ">=0.2.3 <1.0.0", "", ""},
		{"next major", `.upToNextMajor(from: "1.2.3")`, ">=1.2.3 <2.0.0", "", ""},
		{"next minor", `.upToNextMinor(from: "1.2.3")`, ">=1.2.3 <1.3.0", "", ""},
		{"half open", `"1.2.3"..<"2.3.4"`, ">=1.2.3 <2.3.4", "", ""},
		{"closed", `"1.2.3"..."2.3.4"`, ">=1.2.3 <=2.3.4", "", ""},
		{"branch", `branch: "release"`, "release", "release", ""},
		{"legacy branch", `.branch("release")`, "release", "release", ""},
		{"revision", `revision: "abcdef"`, "abcdef", "", "abcdef"},
		{"legacy revision", `.revision("abcdef")`, "abcdef", "", "abcdef"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := `// swift-tools-version: 5.9
import PackageDescription
let package = Package(
    name: "App",
    dependencies: [
        .package(name: "Logging", url: "https://github.com/apple/swift-log.git", ` + tt.declaration + `)
    ]
)`
			result, err := Parse("Package.swift", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if result.Name != "App" || len(result.Dependencies) != 1 {
				t.Fatalf("Parse = %+v", result)
			}
			dep := result.Dependencies[0]
			if dep.Version != tt.want || dep.Source.Branch != tt.branch || dep.Source.Ref != tt.ref {
				t.Errorf("dependency = %+v, want requirement %q, branch %q, ref %q", dep, tt.want, tt.branch, tt.ref)
			}
			if dep.Source.Kind != SourceGit || dep.Source.Value != "https://github.com/apple/swift-log.git" {
				t.Errorf("Source = %+v", dep.Source)
			}
			if dep.PURL != "pkg:swift/github.com/apple/swift-log" || !dep.Direct {
				t.Errorf("dependency identity = %+v", dep)
			}
		})
	}
}

func TestSwiftRegistryManifest(t *testing.T) {
	result, err := Parse("Package.swift", []byte(`import PackageDescription
let package = Package(name: "App", dependencies: [
    .package(id: "apple.swift-log", from: "1.2.3"),
    .package(id: "other.logging", exact: "2.0.0")
])`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 2 {
		t.Fatalf("Dependencies = %+v", result.Dependencies)
	}
	for i, name := range []string{"apple.swift-log", "other.logging"} {
		dep := result.Dependencies[i]
		wantPURL := "pkg:swift/" + strings.Replace(name, ".", "/", 1)
		if dep.Name != name || dep.PURL != wantPURL || dep.Ecosystem != "swift" || !dep.Direct {
			t.Errorf("dependency = %+v, want registry identity %q with PURL %q", dep, name, wantPURL)
		}
		if dep.Source.Kind != SourceRegistry || dep.Source.Value != name {
			t.Errorf("Source = %+v", dep.Source)
		}
	}
	if result.Dependencies[0].Version != ">=1.2.3 <2.0.0" || result.Dependencies[1].Version != "2.0.0" {
		t.Errorf("requirements = %+v", result.Dependencies)
	}
}

func TestSwiftIgnoresCommentsAndStringContents(t *testing.T) {
	content := `// swift-tools-version: 5.9
import PackageDescription
// Package(name: "Fake")
let documentation = """
.package(url: "https://example.com/owner/documentation", from: "1.0.0")
"""
let raw = #".package(url: "https://example.com/owner/raw", from: "1.0.0")"#
let text = ".package(url: \"https://example.com/owner/text\", from: \"1.0.0\")"
/* .package(url: "https://example.com/owner/comment", from: "1.0.0")
   /* nested */
   .package(id: "owner.nested", from: "1.0.0")
*/
let package = Package(name: "App", dependencies: [
    // .package(url: "https://example.com/owner/line", from: "1.0.0")
    .package(
        url: /* between tokens */ "https://github.com/apple/swift-log.git",
        /* requirement */ exact: "1.2.3"
    ),
    .package(url: #"https://github.com/apple/swift-nio.git"#, from: "2.0.0")
])
`
	result, err := Parse("Package.swift", []byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if result.Name != "App" || len(result.Dependencies) != 2 {
		t.Fatalf("Parse = %+v", result)
	}
	if result.Dependencies[0].Name != "github.com/apple/swift-log" || result.Dependencies[0].Version != "1.2.3" {
		t.Errorf("dependency = %+v", result.Dependencies[0])
	}
	if result.Dependencies[1].Name != "github.com/apple/swift-nio" {
		t.Errorf("raw literal dependency = %+v", result.Dependencies[1])
	}
}

func TestSwiftDynamicArguments(t *testing.T) {
	result, err := Parse("Package.swift", []byte(`let package = Package(name: "App", dependencies: [
    .package(url: "https://github.com/" + owner + "/dynamic", exact: "1.0.0"),
    .package(url: "https://github.com/\(owner)/interpolated", from: "1.0.0"),
    .package(url: repository, from: "1.0.0"),
    .package(url: "https://github.com/apple/swift-log.git", exact: version)
])`))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "github.com/apple/swift-log" || result.Dependencies[0].Version != "" {
		t.Fatalf("Dependencies = %+v", result.Dependencies)
	}
}

func TestSwiftVersionSpecificManifest(t *testing.T) {
	for _, filename := range []string{"Package.swift", "Package@swift-5.9.swift", "nested/Package@swift-6.0.swift"} {
		t.Run(filename, func(t *testing.T) {
			content := `let package = Package(name: "App", dependencies: [.package(id: "apple.swift-log", exact: "1.2.3")])`
			result, err := Parse(filename, []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if result.Ecosystem != "swift" || result.Kind != Manifest || len(result.Dependencies) != 1 {
				t.Fatalf("Parse = %+v", result)
			}
		})
	}
}

func TestSwiftResolvedRevisions(t *testing.T) {
	for _, tt := range []struct {
		schema  int
		version string
	}{
		{1, ""}, {1, "1.2.3"},
		{2, ""}, {2, "1.2.3"},
		{3, ""}, {3, "1.2.3"},
	} {
		t.Run(fmt.Sprintf("schema-%d/version-%s", tt.schema, tt.version), func(t *testing.T) {
			revision := strings.Repeat("a", 40)
			branch := "main"
			state := fmt.Sprintf(`{"branch":"main","revision":%q}`, revision)
			if tt.version != "" {
				branch = ""
				state = fmt.Sprintf(`{"revision":%q,"version":%q}`, revision, tt.version)
			}
			pin := fmt.Sprintf(`{"identity":"swift-log","kind":"remoteSourceControl","location":"https://github.com/apple/swift-log.git","state":%s}`, state)
			content := fmt.Sprintf(`{"version":%d,"pins":[%s]}`, tt.schema, pin)
			if tt.schema == 1 {
				content = fmt.Sprintf(`{"version":1,"object":{"pins":[{"package":"Logging","repositoryURL":"https://github.com/apple/swift-log.git","state":%s}]}}`, state)
			}
			result, err := Parse("Package.resolved", []byte(content))
			if err != nil {
				t.Fatal(err)
			}
			if len(result.Dependencies) != 1 {
				t.Fatalf("Dependencies = %+v", result.Dependencies)
			}
			wantVersion := tt.version
			if wantVersion == "" {
				wantVersion = revision
			}
			dep := result.Dependencies[0]
			if dep.Version != wantVersion || dep.Source.Ref != revision || dep.Source.Branch != branch {
				t.Errorf("dependency = %+v", dep)
			}
			if dep.PURL != "pkg:swift/github.com/apple/swift-log@"+wantVersion {
				t.Errorf("PURL = %q", dep.PURL)
			}
			if dep.Source.Kind != SourceGit || dep.Source.Value != "https://github.com/apple/swift-log.git" {
				t.Errorf("Source = %+v", dep.Source)
			}
		})
	}
}
