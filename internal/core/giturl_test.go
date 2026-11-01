package core

import "testing"

func TestGitCoordinate(t *testing.T) {
	tests := map[string]string{
		"https://github.com/apple/swift-argument-parser.git": "github.com/apple/swift-argument-parser",
		"ssh://git@github.com/apple/swift-nio.git":           "github.com/apple/swift-nio",
		"ssh://git@git.example.org:2222/team/repo.git":       "git.example.org:2222/team/repo",
		"git@github.com:apple/swift-log.git":                 "github.com/apple/swift-log",
		"GitLab.com:group/subgroup/repo":                     "gitlab.com/group/subgroup/repo",
		"../local-package":                                   "",
		"./vendor/local.git":                                 "",
		"/srv/git/local.git":                                 "",
		"file:///srv/git/local.git":                          "",
		"C:\\repos\\local":                                   "",
	}

	for rawURL, want := range tests {
		if got := GitCoordinate(rawURL); got != want {
			t.Errorf("GitCoordinate(%q) = %q, want %q", rawURL, got, want)
		}
	}
}
