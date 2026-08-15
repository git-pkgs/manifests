package maven

import (
	"testing"

	"github.com/git-pkgs/manifests/internal/core"
	"github.com/git-pkgs/pom"
)

func TestPOMDeclarations(t *testing.T) {
	content := []byte(`<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
  <parent>
    <groupId>org.example</groupId>
    <artifactId>parent</artifactId>
    <version>1.0.0</version>
  </parent>
  <groupId>org.example</groupId>
  <artifactId>app</artifactId>
  <version>1.0.0</version>
  <dependencies>
    <dependency>
      <groupId>org.example</groupId>
      <artifactId>runtime</artifactId>
      <version>2.0.0</version>
    </dependency>
    <dependency>
      <groupId>org.example</groupId>
      <artifactId>managed</artifactId>
    </dependency>
  </dependencies>
  <dependencyManagement>
    <dependencies>
      <dependency>
        <groupId>org.example</groupId>
        <artifactId>managed</artifactId>
        <version>3.0.0</version>
      </dependency>
    </dependencies>
  </dependencyManagement>
  <build>
    <plugins>
      <plugin>
        <artifactId>maven-compiler-plugin</artifactId>
        <version>4.0.0</version>
        <dependencies>
          <dependency>
            <groupId>org.example</groupId>
            <artifactId>plugin-runtime</artifactId>
            <version>5.0.0</version>
          </dependency>
        </dependencies>
      </plugin>
    </plugins>
    <pluginManagement>
      <plugins>
        <plugin>
          <groupId>org.example</groupId>
          <artifactId>managed-plugin</artifactId>
          <version>6.0.0</version>
          <dependencies>
            <dependency>
              <groupId>org.example</groupId>
              <artifactId>managed-plugin-runtime</artifactId>
              <version>6.1.0</version>
            </dependency>
          </dependencies>
        </plugin>
      </plugins>
    </pluginManagement>
    <extensions>
      <extension>
        <groupId>org.example</groupId>
        <artifactId>extension</artifactId>
        <version>7.0.0</version>
      </extension>
    </extensions>
  </build>
  <profiles>
    <profile>
      <id>release</id>
      <dependencies>
        <dependency>
          <groupId>org.example</groupId>
          <artifactId>profile-runtime</artifactId>
          <version>8.0.0</version>
          <scope>test</scope>
        </dependency>
      </dependencies>
      <dependencyManagement>
        <dependencies>
          <dependency>
            <groupId>org.example</groupId>
            <artifactId>profile-managed</artifactId>
            <version>9.0.0</version>
          </dependency>
        </dependencies>
      </dependencyManagement>
      <build>
        <plugins>
          <plugin>
            <groupId>org.example</groupId>
            <artifactId>profile-plugin</artifactId>
            <version>10.0.0</version>
            <dependencies>
              <dependency>
                <groupId>org.example</groupId>
                <artifactId>profile-plugin-runtime</artifactId>
                <version>10.1.0</version>
              </dependency>
            </dependencies>
          </plugin>
        </plugins>
        <pluginManagement>
          <plugins>
            <plugin>
              <groupId>org.example</groupId>
              <artifactId>profile-managed-plugin</artifactId>
              <version>10.2.0</version>
              <dependencies>
                <dependency>
                  <groupId>org.example</groupId>
                  <artifactId>profile-managed-plugin-runtime</artifactId>
                  <version>10.3.0</version>
                </dependency>
              </dependencies>
            </plugin>
          </plugins>
        </pluginManagement>
        <extensions>
          <extension>
            <groupId>org.example</groupId>
            <artifactId>profile-extension</artifactId>
            <version>11.0.0</version>
          </extension>
        </extensions>
      </build>
    </profile>
  </profiles>
</project>`)

	result, err := (&pomXMLParser{}).Parse("pom.xml", content)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(result.Dependencies) != 2 {
		t.Fatalf("Dependencies has %d entries, want 2: %+v", len(result.Dependencies), result.Dependencies)
	}

	want := map[string]struct {
		name    string
		version string
		scope   core.Scope
	}{
		"parent/org.example:parent":                                    {"org.example:parent", "1.0.0", core.Build},
		"dependencies/org.example:runtime":                             {"org.example:runtime", "2.0.0", core.Runtime},
		"dependencies/org.example:managed":                             {"org.example:managed", "", core.Runtime},
		"dependencyManagement/dependencies/org.example:managed":        {"org.example:managed", "3.0.0", core.Runtime},
		"build/plugins/org.apache.maven.plugins:maven-compiler-plugin": {"org.apache.maven.plugins:maven-compiler-plugin", "4.0.0", core.Build},
		"build/plugins/org.apache.maven.plugins:maven-compiler-plugin/dependencies/org.example:plugin-runtime":                                       {"org.example:plugin-runtime", "5.0.0", core.Build},
		"build/pluginManagement/plugins/org.example:managed-plugin":                                                                                  {"org.example:managed-plugin", "6.0.0", core.Build},
		"build/pluginManagement/plugins/org.example:managed-plugin/dependencies/org.example:managed-plugin-runtime":                                  {"org.example:managed-plugin-runtime", "6.1.0", core.Build},
		"build/extensions/org.example:extension":                                                                                                     {"org.example:extension", "7.0.0", core.Build},
		"profiles/release/dependencies/org.example:profile-runtime":                                                                                  {"org.example:profile-runtime", "8.0.0", core.Test},
		"profiles/release/dependencyManagement/dependencies/org.example:profile-managed":                                                             {"org.example:profile-managed", "9.0.0", core.Runtime},
		"profiles/release/build/plugins/org.example:profile-plugin":                                                                                  {"org.example:profile-plugin", "10.0.0", core.Build},
		"profiles/release/build/plugins/org.example:profile-plugin/dependencies/org.example:profile-plugin-runtime":                                  {"org.example:profile-plugin-runtime", "10.1.0", core.Build},
		"profiles/release/build/pluginManagement/plugins/org.example:profile-managed-plugin":                                                         {"org.example:profile-managed-plugin", "10.2.0", core.Build},
		"profiles/release/build/pluginManagement/plugins/org.example:profile-managed-plugin/dependencies/org.example:profile-managed-plugin-runtime": {"org.example:profile-managed-plugin-runtime", "10.3.0", core.Build},
		"profiles/release/build/extensions/org.example:profile-extension":                                                                            {"org.example:profile-extension", "11.0.0", core.Build},
	}

	if len(result.Declarations) != len(want) {
		t.Fatalf("Declarations has %d entries, want %d: %+v", len(result.Declarations), len(want), result.Declarations)
	}
	for _, declaration := range result.Declarations {
		expected, ok := want[declaration.Location]
		if !ok {
			t.Errorf("unexpected declaration at %q: %+v", declaration.Location, declaration)
			continue
		}
		if declaration.Name != expected.name || declaration.Version != expected.version || declaration.Scope != expected.scope {
			t.Errorf("declaration at %q = %+v, want name %q, version %q, scope %q", declaration.Location, declaration, expected.name, expected.version, expected.scope)
		}
	}
}

func TestPOMDeclarationProfileDefaultsToDefaultID(t *testing.T) {
	content := []byte(`<project><profiles><profile><dependencies><dependency>
  <groupId>org.example</groupId><artifactId>profile-runtime</artifactId><version>1.0.0</version>
</dependency></dependencies></profile></profiles></project>`)
	project, err := pom.ParsePOM(content)
	if err != nil {
		t.Fatalf("ParsePOM: %v", err)
	}
	declarations := parsePOMDeclarations(project)
	if len(declarations) != 1 {
		t.Fatalf("declarations has %d entries, want 1: %+v", len(declarations), declarations)
	}
	if declarations[0].Location != "profiles/default/dependencies/org.example:profile-runtime" {
		t.Errorf("Location = %q, want default profile location", declarations[0].Location)
	}
}

func TestPOMDeclarationLocationIncludesTypeAndClassifier(t *testing.T) {
	content := []byte(`<project><dependencyManagement><dependencies>
  <dependency><groupId>org.example</groupId><artifactId>lib</artifactId><version>1.0.0</version></dependency>
  <dependency><groupId>org.example</groupId><artifactId>lib</artifactId><version>1.0.0</version><type>test-jar</type></dependency>
  <dependency><groupId>org.example</groupId><artifactId>lib</artifactId><version>1.0.0</version><classifier>sources</classifier></dependency>
</dependencies></dependencyManagement></project>`)
	project, err := pom.ParsePOM(content)
	if err != nil {
		t.Fatalf("ParsePOM: %v", err)
	}
	declarations := parsePOMDeclarations(project)
	want := map[string]bool{
		"dependencyManagement/dependencies/org.example:lib":          true,
		"dependencyManagement/dependencies/org.example:lib:test-jar": true,
		"dependencyManagement/dependencies/org.example:lib:sources":  true,
	}
	if len(declarations) != len(want) {
		t.Fatalf("declarations has %d entries, want %d: %+v", len(declarations), len(want), declarations)
	}
	for _, declaration := range declarations {
		if !want[declaration.Location] {
			t.Errorf("unexpected declaration location %q", declaration.Location)
		}
		if declaration.Name != "org.example:lib" {
			t.Errorf("Name = %q, want org.example:lib", declaration.Name)
		}
	}
}

func TestAppendMavenDeclarationSkipsIncompleteCoordinates(t *testing.T) {
	var declarations []core.Declaration
	appendMavenDeclaration(&declarations, "dependencies", "org.example", "", "1.0.0", core.Runtime, "")
	appendMavenDeclaration(&declarations, "dependencies", "", "dependency", "1.0.0", core.Runtime, "")
	appendMavenDeclaration(&declarations, "build/plugins", "", "", "1.0.0", core.Build, defaultMavenPluginGroup)

	if len(declarations) != 0 {
		t.Errorf("declarations = %+v, want no incomplete coordinates", declarations)
	}
}
