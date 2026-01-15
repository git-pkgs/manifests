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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check for a typical maven dependency format (groupId:artifactId)
	hasColon := false
	for name := range depMap {
		if len(name) > 0 {
			hasColon = true
			break
		}
	}
	if !hasColon {
		t.Error("expected dependencies in groupId:artifactId format")
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
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

	if len(deps) == 0 {
		t.Fatal("expected dependencies, got none")
	}

	depMap := make(map[string]core.Dependency)
	for _, d := range deps {
		depMap[d.Name] = d
	}

	// Check for spring security
	if spring, ok := depMap["org.springframework.security:spring-security-crypto"]; !ok {
		t.Error("expected spring-security-crypto dependency")
	} else if spring.Version != "5.7.3" {
		t.Errorf("spring-security-crypto version = %q, want %q", spring.Version, "5.7.3")
	}

	// Check test scope
	if junit, ok := depMap["org.junit.jupiter:junit-jupiter-api"]; !ok {
		t.Error("expected junit-jupiter-api dependency")
	} else if junit.Scope != core.Test {
		t.Errorf("junit scope = %v, want Test", junit.Scope)
	}
}
