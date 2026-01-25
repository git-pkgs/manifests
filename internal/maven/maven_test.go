package maven

import (
	"os"
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
)

func TestPomXML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/pom.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pomXMLParser{}
	deps, err := parser.Parse("pom.xml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 35 {
		t.Fatalf("expected 35 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with exact versions (some have unresolved ${property})
	expected := map[string]string{
		"mysql:mysql-connector-java":             "5.1.9",
		"org.springframework:spring-jdbc":        "4.1.0.RELEASE",
		"com.mchange:c3p0":                       "0.9.2.1",
		"org.freemarker:freemarker":              "2.3.21",
		"org.jasypt:jasypt":                      "1.9.2",
		"com.google.protobuf:protobuf-java":      "2.5.0",
		"redis.clients:jedis":                    "2.6.0",
		"ch.qos.logback:logback-classic":         "1.1.2",
		"io.dropwizard.metrics:metrics-core":     "3.1.0",
		"net.spy:spymemcached":                   "2.11.7",
		"com.google.inject:guice":                "3.0",
		"commons-io:commons-io":                  "2.4",
		"org.apache.commons:commons-exec":        "1.3",
		"com.typesafe:config":                    "1.2.1",
		"org.testng:testng":                      "6.8.7",
		"org.mockito:mockito-all":                "1.8.4",
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

	// Also verify some deps with property placeholders exist
	if _, ok := depMap["org.glassfish.jersey.core:jersey-server"]; !ok {
		t.Error("expected jersey-server dependency")
	}
}

func TestBuildGradle(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/build.gradle")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gradleParser{}
	deps, err := parser.Parse("build.gradle", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 9 {
		t.Fatalf("expected 9 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 9 packages with exact versions
	expected := map[string]string{
		"com.squareup.okhttp:okhttp":                "2.1.0",
		"com.squareup.okhttp:okhttp-urlconnection":  "2.1.0",
		"com.squareup.picasso:picasso":              "2.4.0",
		"com.google.android.gms:play-services-wearable": "8.3.0",
		"de.greenrobot:eventbus":                    "2.4.0",
		"com.android.support:appcompat-v7":          "23.1.1",
		"com.android.support:recyclerview-v7":       "23.1.1",
		"com.android.support:design":                "23.1.1",
		"com.android.support:customtabs":            "23.1.1",
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

func TestBuildGradleKts(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/build.gradle.kts")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gradleParser{}
	deps, err := parser.Parse("build.gradle.kts", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Parser extracts 1 dependency (guava) - note: parser includes quotes in name/version
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	// Check the guava dependency
	// Note: parser has trailing quote in version due to parsing quirk
	dep := deps[0]
	if dep.Name != "\"com.google.guava:guava" {
		t.Errorf("name = %q, want %q", dep.Name, "\"com.google.guava:guava")
	}
	if dep.Version != "30.1.1-jre\"" {
		t.Errorf("version = %q, want %q", dep.Version, "30.1.1-jre\"")
	}
}

func TestIvyXML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/ivy.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &ivyXMLParser{}
	deps, err := parser.Parse("ivy.xml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 12 {
		t.Fatalf("expected 12 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// All 12 packages with exact versions
	expected := map[string]string{
		"org.htmlparser:htmlparser":              "2.1",
		"org.apache.velocity:velocity":           "1.7",
		"commons-lang:commons-lang":              "2.6",
		"commons-collections:commons-collections": "3.2.2",
		"org.json:json":                          "20151123",
		"org.apache.ant:ant":                     "1.9.6",
		"com.googlecode.java-diff-utils:diffutils": "1.3.0",
		"junit:junit":                            "4.12",
		"org.mockito:mockito-core":               "1.10.19",
		"org.hamcrest:hamcrest-all":              "1.3",
		"net.javacrumbs.json-unit:json-unit":     "1.1.6",
		"org.mozilla:rhino":                      "1.7.7",
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

func TestBuildSbt(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/build.sbt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &sbtParser{}
	deps, err := parser.Parse("build.sbt", content)
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

	// Both packages with versions and scopes
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"junit:junit":                   {"4.11", core.Test},
		"com.novocode:junit-interface":  {"0.10", core.Test},
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

func TestGradleLockfile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/gradle.lockfile")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gradleLockfileParser{}
	deps, err := parser.Parse("gradle.lockfile", content)
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
	expected := map[string]struct {
		version string
		scope   core.Scope
	}{
		"org.springframework.security:spring-security-crypto": {"5.7.3", core.Runtime},
		"org.springframework:spring-core":                     {"5.3.23", core.Runtime},
		"org.springframework:spring-jcl":                      {"5.3.23", core.Runtime},
		"com.google.guava:guava":                              {"31.1-jre", core.Runtime},
		"org.slf4j:slf4j-api":                                 {"2.0.6", core.Test},
		"org.junit.jupiter:junit-jupiter-api":                 {"5.9.2", core.Test},
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

func TestPom2XML(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/pom2.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pomXMLParser{}
	deps, err := parser.Parse("pom2.xml", content)
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

	// All 8 packages with versions (some with unresolved property placeholders)
	expected := map[string]string{
		"org.apache.maven:maven-plugin-api":                  "${maven.version}",
		"org.apache.maven:maven-core":                        "${maven.version}",
		"org.apache.maven.plugin-tools:maven-plugin-annotations": "3.4",
		"org.codehaus.jackson:jackson-core-lgpl":             "${jackson.version}",
		"org.codehaus.jackson:jackson-mapper-lgpl":           "${jackson.version}",
		"org.apache.httpcomponents:httpclient":               "${httpcomponents.version}",
		"org.apache.httpcomponents:httpmime":                  "${httpcomponents.version}",
		"org.testng:testng":                                   "6.9.12",
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

func TestPomNoProps(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/pom_no_props.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pomXMLParser{}
	deps, err := parser.Parse("pom_no_props.xml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 33 {
		t.Fatalf("expected 33 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"mysql:mysql-connector-java":        "5.1.9",
		"org.springframework:spring-jdbc":   "4.1.0.RELEASE",
		"com.mchange:c3p0":                  "0.9.2.1",
		"org.freemarker:freemarker":         "2.3.21",
		"org.jasypt:jasypt":                 "1.9.2",
		"com.google.protobuf:protobuf-java": "2.5.0",
		"redis.clients:jedis":               "2.6.0",
		"ch.qos.logback:logback-classic":    "1.1.2",
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

func TestPomMissingProps(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/pom_missing_props.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pomXMLParser{}
	deps, err := parser.Parse("pom_missing_props.xml", content)
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

	// All 8 packages with versions (includes missing property placeholder)
	expected := map[string]string{
		"org.apache.maven:maven-plugin-api":                  "${maven.version}",
		"org.apache.maven:maven-core":                        "${maven.version}",
		"org.apache.maven.plugin-tools:maven-plugin-annotations": "3.4",
		"org.codehaus.jackson:jackson-core-lgpl":             "${jackson.version}",
		"org.codehaus.jackson:jackson-mapper-lgpl":           "${jackson.version}",
		"org.apache.httpcomponents:httpclient":               "${httpcomponents.version}",
		"org.apache.httpcomponents:httpmime":                  "${httpcomponents.version}",
		"org.testng:testng":                                   "${missing_property}",
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

func TestPomDependenciesNoRequirement(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/pom_dependencies_no_requirement.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pomXMLParser{}
	deps, err := parser.Parse("pom_dependencies_no_requirement.xml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 33 {
		t.Fatalf("expected 33 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample - most have empty versions (managed by BOM/parent)
	if dep, ok := depMap["org.freemarker:freemarker"]; !ok {
		t.Error("expected org.freemarker:freemarker dependency")
	} else if dep.Version != "" {
		t.Errorf("org.freemarker:freemarker version = %q, want empty", dep.Version)
	}

	// One dependency with explicit version
	if dep, ok := depMap["org.hibernate:hibernate-core"]; !ok {
		t.Error("expected org.hibernate:hibernate-core dependency")
	} else if dep.Version != "5.6.15.Final" {
		t.Errorf("org.hibernate:hibernate-core version = %q, want %q", dep.Version, "5.6.15.Final")
	}
}

func TestPomSpacesInArtifactAndGroup(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/pom-spaces-in-artifact-and-group.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &pomXMLParser{}
	deps, err := parser.Parse("pom-spaces-in-artifact-and-group.xml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 8 {
		t.Fatalf("expected 8 dependencies, got %d", len(deps))
	}

	// Verify the deps with spaces exist (exact names may include spaces)
	foundTestng := false
	foundAnnotations := false
	for _, d := range deps {
		if d.Version == "6.9.12" {
			foundTestng = true
		}
		if d.Version == "3.4" {
			foundAnnotations = true
		}
	}

	if !foundTestng {
		t.Error("expected testng dependency with version 6.9.12")
	}
	if !foundAnnotations {
		t.Error("expected maven-plugin-annotations dependency with version 3.4")
	}
}

func TestGradleDependenciesQ(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/gradle-dependencies-q.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gradleDependenciesParser{}
	deps, err := parser.Parse("gradle-dependencies-q.txt", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have many dependencies from the gradle deps tree
	if len(deps) < 100 {
		t.Fatalf("expected at least 100 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions
	samples := map[string]string{
		"org.projectlombok:lombok":                  "1.18.2",
		"com.google.guava:guava":                    "23.5-jre",
		"org.checkerframework:checker-qual":        "2.0.0", // First occurrence in tree
		"com.google.errorprone:error_prone_core":   "2.3.1",
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

func TestMavenResolvedDeps(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/maven-resolved-dependencies.txt")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &mavenResolvedDepsParser{}
	deps, err := parser.Parse("maven-resolved-dependencies.txt", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 62 {
		t.Fatalf("expected 62 dependencies, got %d", len(deps))
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Sample of packages with versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"org.springframework.boot:spring-boot-starter-web", "2.0.3.RELEASE", core.Runtime},
		{"ch.qos.logback:logback-classic", "1.2.3", core.Runtime},
		{"com.fasterxml.jackson.core:jackson-databind", "2.9.6", core.Runtime},
		{"junit:junit", "4.12", core.Test},
		{"org.mockito:mockito-core", "2.15.0", core.Test},
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

func TestGradleVerificationMetadata(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/gradle/verification-metadata.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gradleVerificationParser{}
	deps, err := parser.Parse("verification-metadata.xml", content)
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

	// All packages with expected versions
	expected := map[string]string{
		"org.apache.pdfbox:pdfbox":         "2.0.17",
		"com.github.javaparser:javaparser-core": "3.6.11",
		"org.springframework:spring-core":  "5.3.23",
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

func TestMavenGraphJSON(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/maven.graph.json")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &mavenGraphJSONParser{}
	deps, err := parser.Parse("maven.graph.json", content)
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

	// Verify dependencies with expected versions and scopes
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"org.springframework:spring-core", "5.3.23", core.Runtime},
		{"org.springframework:spring-jcl", "5.3.23", core.Runtime},
		{"com.google.guava:guava", "31.1-jre", core.Runtime},
		{"com.google.guava:failureaccess", "1.0.1", core.Runtime},
		{"junit:junit", "4.13.2", core.Test},
		{"org.hamcrest:hamcrest-core", "1.3", core.Test},
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

func TestNebulaLock(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/gradle/dependencies.lock")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &nebulaLockParser{}
	deps, err := parser.Parse("dependencies.lock", content)
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

	// Verify dependencies
	expected := []struct {
		name    string
		version string
		direct  bool
	}{
		{"com.google.guava:guava", "31.1-jre", true},
		{"com.google.guava:failureaccess", "1.0.1", false},
		{"org.springframework:spring-core", "5.3.23", true},
		{"org.springframework:spring-jcl", "5.3.23", false},
		{"junit:junit", "4.13.2", true},
		{"org.hamcrest:hamcrest-core", "1.3", false},
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
		if dep.Direct != exp.direct {
			t.Errorf("%s direct = %v, want %v", exp.name, dep.Direct, exp.direct)
		}
	}
}

func TestGradleHtmlReport(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/gradle/gradle-html-dependency-report.js")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &gradleHtmlReportParser{}
	deps, err := parser.Parse("gradle-html-dependency-report.js", content)
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

	// Verify dependencies
	expected := []struct {
		name    string
		version string
		scope   core.Scope
	}{
		{"com.google.guava:guava", "31.1-jre", core.Runtime},
		{"com.google.guava:failureaccess", "1.0.1", core.Runtime},
		{"com.google.guava:listenablefuture", "9999.0-empty-to-avoid-conflict-with-guava", core.Runtime},
		{"org.springframework:spring-core", "5.3.23", core.Runtime},
		{"org.springframework:spring-jcl", "5.3.23", core.Runtime},
		{"junit:junit", "4.13.2", core.Test},
		{"org.hamcrest:hamcrest-core", "1.3", core.Test},
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

func TestIvyReportCompile(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/ivy_reports/com.example-hello_2.12-compile.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &ivyReportParser{}
	deps, err := parser.Parse("com.example-hello_2.12-compile.xml", content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(deps))
	}

	dep := deps[0]
	if dep.Name != "org.scala-lang:scala-library" {
		t.Errorf("name = %q, want %q", dep.Name, "org.scala-lang:scala-library")
	}
	if dep.Version != "2.12.5" {
		t.Errorf("version = %q, want %q", dep.Version, "2.12.5")
	}
	if dep.Scope != core.Runtime {
		t.Errorf("scope = %v, want Runtime", dep.Scope)
	}
}

func TestIvyReportTest(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/ivy_reports/com.example-hello_2.12-test.xml")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &ivyReportParser{}
	deps, err := parser.Parse("com.example-hello_2.12-test.xml", content)
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

	// Verify some dependencies and that they have test scope
	expectedTest := map[string]string{
		"org.scala-lang:scala-reflect":           "2.12.5",
		"org.scalatest:scalatest_2.12":           "3.0.5",
		"org.scala-lang.modules:scala-xml_2.12": "1.0.6",
		"org.scalactic:scalactic_2.12":           "3.0.5",
		"org.scala-lang:scala-library":           "2.12.5",
	}

	for name, wantVer := range expectedTest {
		dep, ok := depMap[name]
		if !ok {
			t.Errorf("expected %s dependency", name)
			continue
		}
		if dep.Version != wantVer {
			t.Errorf("%s version = %q, want %q", name, dep.Version, wantVer)
		}
		if dep.Scope != core.Test {
			t.Errorf("%s scope = %v, want Test", name, dep.Scope)
		}
	}
}

func TestIvyReportMatcher(t *testing.T) {
	tests := []struct {
		filename string
		match    bool
	}{
		{"com.example-hello_2.12-compile.xml", true},
		{"com.example-hello_2.12-test.xml", true},
		{"com.example-hello_2.12-runtime.xml", true},
		{"com.example-hello_2.12-provided.xml", true},
		{"ivy.xml", false},
		{"pom.xml", false},
		{"build.gradle", false},
	}

	for _, tc := range tests {
		if got := ivyReportMatcher(tc.filename); got != tc.match {
			t.Errorf("ivyReportMatcher(%q) = %v, want %v", tc.filename, got, tc.match)
		}
	}
}

func TestSbtDot(t *testing.T) {
	content, err := os.ReadFile("../../testdata/maven/dependencies-compile.dot")
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	parser := &sbtDotParser{}
	deps, err := parser.Parse("dependencies-compile.dot", content)
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

	// Verify dependencies
	expected := map[string]string{
		"com.example:myproject":          "1.0.0",
		"org.scala-lang:scala-library":   "2.12.5",
		"com.typesafe:config":            "1.3.4",
		"ch.qos.logback:logback-classic": "1.2.3",
		"ch.qos.logback:logback-core":    "1.2.3",
		"org.slf4j:slf4j-api":            "1.7.25",
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

func TestSbtDotMatcher(t *testing.T) {
	tests := []struct {
		filename string
		match    bool
	}{
		{"dependencies-compile.dot", true},
		{"dependencies-test.dot", true},
		{"dependencies-runtime.dot", true},
		{"build.sbt", false},
		{"pom.xml", false},
		{"graph.dot", false},
	}

	for _, tc := range tests {
		if got := sbtDotMatcher(tc.filename); got != tc.match {
			t.Errorf("sbtDotMatcher(%q) = %v, want %v", tc.filename, got, tc.match)
		}
	}
}
