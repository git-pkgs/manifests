package manifests

import (
	"testing"
)

func TestGitHubActionsDockerPURLs(t *testing.T) {
	content := []byte(`name: fixture
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    container: ghcr.io/example/job:1
    services:
      database:
        image: postgres:16
    steps:
      - uses: actions/checkout@v4
      - uses: docker://ghcr.io/example/step:2
`)
	result, err := Parse(".github/workflows/fixture.yml", content)
	if err != nil {
		t.Fatal(err)
	}
	if result.Ecosystem != "github-actions" || result.Kind != Manifest {
		t.Fatalf("file identity = %q %q, want github-actions manifest", result.Ecosystem, result.Kind)
	}
	want := map[string]struct {
		version, ecosystem, purl string
	}{
		"actions/checkout":              {"v4", "github-actions", "pkg:githubactions/actions/checkout"},
		"docker://ghcr.io/example/step": {"2", "docker", "pkg:docker/ghcr.io/example/step"},
		"docker://ghcr.io/example/job":  {"1", "docker", "pkg:docker/ghcr.io/example/job"},
		"docker://postgres":             {"16", "docker", "pkg:docker/postgres"},
	}
	if len(result.Dependencies) != len(want) {
		t.Fatalf("dependencies = %+v, want %d entries", result.Dependencies, len(want))
	}
	for _, dep := range result.Dependencies {
		expected, ok := want[dep.Name]
		if !ok {
			t.Errorf("unexpected dependency %+v", dep)
			continue
		}
		if dep.Version != expected.version || dep.Ecosystem != expected.ecosystem || dep.PURL != expected.purl ||
			dep.Scope != Runtime || !dep.Direct {
			t.Errorf("dependency = %+v, want version %q, ecosystem %q, PURL %q, direct runtime",
				dep, expected.version, expected.ecosystem, expected.purl)
		}
		delete(want, dep.Name)
	}
	if len(want) != 0 {
		t.Errorf("missing dependencies: %+v", want)
	}
}

func TestGitHubActionsDockerReferences(t *testing.T) {
	const digest = "sha256:7b65413af120ec5328077775022c78101f103258a1876ec2f83890bce416e896"
	content := []byte(`jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: registry.example:5000/team/image:stable@` + digest + `
    services:
      duplicate:
        image: registry.example:5000/team/image:stable@` + digest + `
      untagged:
        image: ghcr.io/example/untagged
    steps:
      - uses: docker://registry.example:5000/team/image:stable@` + digest + `
      - uses: docker://registry.example:5000/team/image:2
`)
	result, err := Parse(".github/workflows/fixture.yaml", content)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		digest:   "pkg:docker/registry.example:5000/team/image",
		"2":      "pkg:docker/registry.example:5000/team/image",
		"latest": "pkg:docker/ghcr.io/example/untagged",
	}
	if len(result.Dependencies) != len(want) {
		t.Fatalf("dependencies = %+v, want %d entries", result.Dependencies, len(want))
	}
	for _, dep := range result.Dependencies {
		expected, ok := want[dep.Version]
		if !ok || dep.PURL != expected || dep.Ecosystem != "docker" {
			t.Errorf("unexpected dependency %+v, want PURL %q and docker ecosystem", dep, expected)
		}
		delete(want, dep.Version)
	}
	if len(want) != 0 {
		t.Errorf("missing versions: %+v", want)
	}
}
