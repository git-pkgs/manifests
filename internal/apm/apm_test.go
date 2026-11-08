package apm

import "testing"

func TestGitCoordinate(t *testing.T) {
	tests := []struct {
		locator, hostType, repo, virtual string
	}{
		{"owner/repo", "", "github.com/owner/repo", ""},
		{"Owner/Repo/skills/x", "", "github.com/owner/repo", "skills/x"},
		{"github.com/owner/repo", "", "github.com/owner/repo", ""},
		{"https://github.com/owner/repo.git", "", "github.com/owner/repo", ""},
		{"git@github.com:owner/repo.git", "", "github.com/owner/repo", ""},
		{"ssh://git@bitbucket.example.com:7999/project/repo.git", "", "bitbucket.example.com:7999/project/repo", ""},
		{"git.internal:8443/team/repo", "", "git.internal:8443/team/repo", ""},
		{"git.internal:443/team/repo", "", "git.internal/team/repo", ""},
		{"gitlab.com/group/sub/repo", "", "gitlab.com/group/sub/repo", ""},
		{"gitlab.com/group/repo/.github/skills/x", "", "gitlab.com/group/repo", ".github/skills/x"},
		{"code.example.com/team/Repo", "gitlab", "code.example.com/team/Repo", ""},
		{"dev.azure.com/org/project/_git/repo", "", "dev.azure.com/org/project/repo", ""},
		{"dev.azure.com/org/project/_git/repo/skills", "", "dev.azure.com/org/project/repo", "skills"},
		{"bitbucket.org/team/repo/prompts/review.prompt.md", "", "bitbucket.org/team/repo", "prompts/review.prompt.md"},
		{"./packages/local", "", "", ""},
		{"parent", "", "", ""},
		{"single", "", "", ""},
	}
	for _, tt := range tests {
		repo, virtual := gitCoordinate(tt.locator, tt.hostType)
		if repo != tt.repo || virtual != tt.virtual {
			t.Errorf("gitCoordinate(%q) = %q, %q; want %q, %q", tt.locator, repo, virtual, tt.repo, tt.virtual)
		}
	}
}

func TestGitDependencyParentIsSourceOnly(t *testing.T) {
	dep := gitDependency("parent", "skills/shared", "", "")
	if !dep.NoPURL || dep.PURL != "" || dep.Name != "parent/skills/shared" {
		t.Errorf("gitDependency(parent) = %+v, want a source-only entry", dep)
	}
}
