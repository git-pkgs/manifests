package pypi

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	// pip-inspect.deplock - lockfile (pip inspect output, as saved by DepLock)
	core.Register("pypi", core.Lockfile, &pipInspectParser{}, core.ExactMatch("pip-inspect.deplock"))
}

// pipInspectParser parses the JSON report written by pip inspect.
type pipInspectParser struct{}

type pipInspectReport struct {
	Installed []struct {
		Metadata struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"metadata"`
		Requested bool                 `json:"requested"`
		DirectURL *pipInspectDirectURL `json:"direct_url"`
	} `json:"installed"`
}

// pipInspectDirectURL is the PEP 610 direct_url.json record of a package
// installed from a URL, a VCS checkout or a local directory.
type pipInspectDirectURL struct {
	URL     string `json:"url"`
	VCSInfo *struct {
		VCS      string `json:"vcs"`
		CommitID string `json:"commit_id"`
	} `json:"vcs_info"`
	ArchiveInfo *struct {
		Hash   string            `json:"hash"`
		Hashes map[string]string `json:"hashes"`
	} `json:"archive_info"`
	DirInfo *struct{} `json:"dir_info"`
}

func (p *pipInspectParser) Parse(filename string, content []byte) (*core.Result, error) {
	var report pipInspectReport
	if err := json.Unmarshal(content, &report); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, entry := range report.Installed {
		if entry.Metadata.Name == "" {
			continue
		}
		dep := core.Dependency{
			Name:    entry.Metadata.Name,
			Version: entry.Metadata.Version,
			Scope:   core.Runtime,
			Direct:  entry.Requested,
		}
		if direct := entry.DirectURL; direct != nil {
			dep.Source = pipInspectSource(direct)
			dep.Integrity = pipInspectIntegrity(direct)
		}
		deps = append(deps, dep)
	}

	return &core.Result{Dependencies: deps}, nil
}

func pipInspectSource(direct *pipInspectDirectURL) core.Source {
	switch {
	case direct.URL == "":
		return core.Source{}
	case direct.VCSInfo != nil && direct.VCSInfo.VCS == "git":
		return core.Source{Kind: core.SourceGit, Value: direct.URL, Ref: direct.VCSInfo.CommitID}
	case direct.DirInfo != nil:
		if parsed, err := url.Parse(direct.URL); err == nil && parsed.Scheme == "file" {
			return core.Source{Kind: core.SourcePath, Value: parsed.Path}
		}
		return core.Source{Kind: core.SourcePath, Value: direct.URL}
	default:
		return core.Source{Kind: core.SourceURL, Value: direct.URL}
	}
}

// pipInspectIntegrity returns the archive sha256 in the sha256-<hex> form
// used by the other pypi lockfiles, falling back to the legacy hash field.
func pipInspectIntegrity(direct *pipInspectDirectURL) string {
	if direct.ArchiveInfo == nil {
		return ""
	}
	if sha := direct.ArchiveInfo.Hashes["sha256"]; sha != "" {
		return "sha256-" + sha
	}
	if sha, ok := strings.CutPrefix(direct.ArchiveInfo.Hash, "sha256="); ok && sha != "" {
		return "sha256-" + sha
	}
	return ""
}
