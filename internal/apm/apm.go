// Package apm parses Microsoft APM (Agent Package Manager) manifests and lockfiles.
package apm

import (
	"cmp"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/git-pkgs/purl"
	"go.yaml.in/yaml/v3"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("apm", core.Manifest, &manifestParser{}, core.ExactMatch("apm.yml"))
	core.Register("apm", core.Lockfile, &lockParser{}, core.ExactMatch("apm.lock.yaml", "apm.lock"))
}

const (
	defaultHost = "github.com"
	selfRepoURL = "<self>"

	// Artifact roles for lockfile hashes.
	artifactArchive  = "archive"
	artifactTree     = "tree"
	artifactDeployed = "deployed:"
)

// scpLocator matches user@host:path SSH shorthand.
var scpLocator = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.+-]*@([^:/]+):(.+)$`)

// virtualFileExtensions mark single-file virtual packages.
var virtualFileExtensions = []string{".prompt.md", ".instructions.md", ".agent.md"}

// virtualRoots start a virtual path inside a GitLab repository.
var virtualRoots = []string{".agents", ".claude", ".codex", ".cursor", ".gemini", ".github", ".kiro", ".windsurf"}

type manifestParser struct{}

type manifest struct {
	Name            string           `yaml:"name"`
	Version         string           `yaml:"version"`
	License         string           `yaml:"license"`
	Scripts         map[string]any   `yaml:"scripts"`
	Registries      map[string]any   `yaml:"registries"`
	Dependencies    dependencyGroups `yaml:"dependencies"`
	DevDependencies dependencyGroups `yaml:"devDependencies"`
}

type dependencyGroups struct {
	APM []any `yaml:"apm"`
	MCP []any `yaml:"mcp"`
	LSP []any `yaml:"lsp"`
}

func (p *manifestParser) Parse(filename string, content []byte) (*core.Result, error) {
	var m manifest
	if err := yaml.Unmarshal(content, &m); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, group := range []struct {
		deps  dependencyGroups
		scope core.Scope
	}{{m.Dependencies, core.Runtime}, {m.DevDependencies, core.Development}} {
		for _, entry := range group.deps.APM {
			if dep, ok := apmDependency(entry, m.Registries); ok {
				deps = append(deps, withScope(dep, group.scope))
			}
		}
		for _, entry := range group.deps.MCP {
			if dep, ok := mcpDependency(entry); ok {
				deps = append(deps, withScope(dep, group.scope))
			}
		}
		for _, entry := range group.deps.LSP {
			if dep, ok := lspDependency(entry); ok {
				deps = append(deps, withScope(dep, group.scope))
			}
		}
	}

	result := &core.Result{
		Name:         m.Name,
		Version:      m.Version,
		Dependencies: deps,
		Scripts:      core.StringScripts(m.Scripts),
	}
	if m.License != "" {
		result.Licenses = []string{m.License}
	}
	return result, nil
}

func withScope(dep core.Dependency, scope core.Scope) core.Dependency {
	dep.Scope = scope
	dep.Direct = true
	return dep
}

// apmDependency converts a dependencies.apm string or object entry.
func apmDependency(entry any, registries map[string]any) (core.Dependency, bool) {
	defaultRegistry, _ := registries["default"].(string)
	switch e := entry.(type) {
	case string:
		s := strings.TrimSpace(e)
		if s == "" {
			return core.Dependency{}, false
		}
		if isLocalPath(s) {
			return localDependency(s, ""), true
		}
		locator, ref, _ := strings.Cut(s, "#")
		// Plain shorthand with a selector routes to the default registry.
		if defaultRegistry != "" && ref != "" && !isHostQualified(locator) && !isURL(locator) {
			return registryDependency(locator, "", ref, registryURL(registries, defaultRegistry)), true
		}
		return gitDependency(locator, "", ref, ""), true
	case map[string]any:
		get := func(key string) string { s, _ := e[key].(string); return s }
		switch {
		case get("marketplace") != "":
			dep := core.Dependency{Name: get("name"), Version: get("version"), NoPURL: true}
			dep.Source = core.Source{Kind: core.SourceRegistry, Value: get("marketplace")}
			return dep, dep.Name != ""
		case get("id") != "" || get("registry") != "":
			id := get("id")
			registry := registryURL(registries, cmp.Or(get("registry"), defaultRegistry))
			return registryDependency(id, get("path"), get("version"), registry), id != ""
		case get("git") != "":
			return gitDependency(get("git"), get("path"), get("ref"), get("type")), true
		case get("path") != "":
			return localDependency(get("path"), ""), true
		}
	}
	return core.Dependency{}, false
}

// registryURL resolves a registry name through apm.yml, keeping names it does not declare.
func registryURL(registries map[string]any, name string) string {
	switch entry := registries[name].(type) {
	case string:
		if name != "default" {
			return entry
		}
	case map[string]any:
		if u, _ := entry["url"].(string); u != "" {
			return u
		}
	}
	return name
}

// mcpDependency converts a dependencies.mcp entry, which has no PURL.
func mcpDependency(entry any) (core.Dependency, bool) {
	dep := core.Dependency{NoPURL: true}
	switch e := entry.(type) {
	case string:
		dep.Name = strings.TrimSpace(e)
		dep.Source = core.Source{Kind: core.SourceRegistry}
	case map[string]any:
		dep.Name, _ = e["name"].(string)
		dep.Version, _ = e["version"].(string)
		switch registry := e["registry"].(type) {
		case bool:
			if !registry {
				if u, _ := e["url"].(string); u != "" {
					dep.Source = core.Source{Kind: core.SourceURL, Value: u}
				}
				break
			}
			dep.Source = core.Source{Kind: core.SourceRegistry}
		case string:
			dep.Source = core.Source{Kind: core.SourceRegistry, Value: registry}
		default:
			dep.Source = core.Source{Kind: core.SourceRegistry}
		}
	}
	return dep, dep.Name != ""
}

// lspDependency converts a dependencies.lsp entry, which has no PURL.
func lspDependency(entry any) (core.Dependency, bool) {
	dep := core.Dependency{NoPURL: true}
	switch e := entry.(type) {
	case string:
		dep.Name = strings.TrimSpace(e)
	case map[string]any:
		dep.Name, _ = e["name"].(string)
	}
	return dep, dep.Name != ""
}

func localDependency(path, version string) core.Dependency {
	return core.Dependency{
		Name:    path,
		Version: version,
		NoPURL:  true,
		Source:  core.Source{Kind: core.SourcePath, Value: path},
	}
}

func registryDependency(id, subpath, version, registry string) core.Dependency {
	name := id
	if subpath != "" {
		name += "/" + strings.Trim(subpath, "/")
	}
	return core.Dependency{
		Name:    name,
		Version: version,
		NoPURL:  true,
		Source:  core.Source{Kind: core.SourceRegistry, Value: registry},
	}
}

// gitDependency identifies a Git source by host, repository and optional subpath.
func gitDependency(locator, subpath, ref, hostType string) core.Dependency {
	dep := core.Dependency{
		Version: ref,
		Source:  core.Source{Kind: core.SourceGit, Value: locator, Ref: ref},
	}
	repo, virtual := gitCoordinate(locator, hostType)
	virtual = joinPath(virtual, subpath)
	if repo == "" {
		dep.Name = joinPath(locator, subpath)
		dep.NoPURL = true
		return dep
	}
	dep.Name = joinPath(repo, virtual)
	dep.Ecosystem = "git"
	dep.PURL = gitPURL(repo, "", virtual)
	return dep
}

// gitCoordinate splits a Git locator into host/repository and a virtual subpath.
func gitCoordinate(locator, hostType string) (repo, virtual string) {
	host, path := splitLocator(strings.TrimSpace(locator))
	if host == "" {
		return "", ""
	}
	segments, hadGitMarker := repoSegments(host, path)
	count := repoSegmentCount(host, hostType, segments, hadGitMarker)
	if count < 2 || len(segments) < count {
		return "", ""
	}
	return host + "/" + repoPathCase(host, strings.Join(segments[:count], "/")), strings.Join(segments[count:], "/")
}

// repoSegments splits a repository path, dropping Azure DevOps markers and a .git suffix.
func repoSegments(host, path string) (segments []string, hadGitMarker bool) {
	segments = slices.DeleteFunc(strings.Split(path, "/"), func(s string) bool { return s == "" })
	if isAzureDevOpsHost(host) {
		if host == "ssh.dev.azure.com" && len(segments) > 0 && segments[0] == "v3" {
			segments = segments[1:]
		}
		hadGitMarker = slices.Contains(segments, "_git")
		segments = slices.DeleteFunc(segments, func(s string) bool { return s == "_git" })
	}
	if len(segments) > 0 {
		last := len(segments) - 1
		segments[last] = strings.TrimSuffix(segments[last], ".git")
	}
	return segments, hadGitMarker
}

// repoPathCase lowercases GitHub repository paths, which APM treats as case-insensitive.
func repoPathCase(host, path string) string {
	if isGitHubHost(host) {
		return strings.ToLower(path)
	}
	return path
}

func isGitHubHost(host string) bool {
	return host == defaultHost || strings.HasSuffix(host, ".ghe.com")
}

func isAzureDevOpsHost(host string) bool {
	return host == "dev.azure.com" || host == "ssh.dev.azure.com" || strings.HasSuffix(host, ".visualstudio.com")
}

// splitLocator returns the lowercased host and path of a Git locator.
func splitLocator(locator string) (host, path string) {
	switch {
	case strings.Contains(locator, "://"):
		parsed, err := url.Parse(locator)
		if err != nil || parsed.Hostname() == "" || parsed.Scheme == "file" {
			return "", ""
		}
		host = strings.ToLower(parsed.Hostname())
		if parsed.Port() != "" && parsed.Port() != "443" {
			host += ":" + parsed.Port()
		}
		return host, parsed.Path
	case scpLocator.MatchString(locator):
		m := scpLocator.FindStringSubmatch(locator)
		return strings.ToLower(m[1]), m[2]
	case isHostQualified(locator):
		hostPart, path, _ := strings.Cut(locator, "/")
		hostPart = strings.ToLower(strings.TrimSuffix(hostPart, ":443"))
		return hostPart, path
	case isLocalPath(locator) || locator == "parent":
		return "", ""
	default:
		return defaultHost, locator
	}
}

// repoSegmentCount mirrors APM's per-host repository/subpath split.
func repoSegmentCount(host, hostType string, segments []string, hadGitMarker bool) int {
	switch {
	case isGitHubHost(host):
		return 2
	case strings.HasSuffix(host, ".visualstudio.com"):
		// The organization is in the subdomain.
		if hadGitMarker || len(segments) == 2 {
			return 2
		}
		return 3
	case isAzureDevOpsHost(host):
		return 3
	case hostType == "gitlab" || host == "gitlab.com":
		if start := virtualStart(segments); start > 0 {
			return start
		}
		return len(segments)
	case hasVirtualFile(segments) || slices.Contains(segments, "collections"):
		return 2
	default:
		return len(segments)
	}
}

// virtualStart returns where a virtual path begins, or 0.
func virtualStart(segments []string) int {
	for i := 2; i < len(segments); i++ {
		if slices.Contains(virtualRoots, segments[i]) {
			return i
		}
	}
	for i := 2; i < len(segments); i++ {
		for _, ext := range virtualFileExtensions {
			if strings.HasSuffix(segments[i], ext) {
				return i
			}
		}
	}
	return 0
}

func hasVirtualFile(segments []string) bool {
	return slices.ContainsFunc(segments, func(s string) bool {
		return slices.ContainsFunc(virtualFileExtensions, func(ext string) bool { return strings.HasSuffix(s, ext) })
	})
}

// isHostQualified reports whether a shorthand starts with a host such as gitlab.com.
func isHostQualified(locator string) bool {
	first, rest, ok := strings.Cut(locator, "/")
	host, _, _ := strings.Cut(first, ":")
	return ok && strings.Contains(host, ".") && strings.Count(rest, "/") >= 1 && !isLocalPath(locator)
}

func isURL(locator string) bool {
	return strings.Contains(locator, "://") || scpLocator.MatchString(locator)
}

func isLocalPath(s string) bool {
	for _, prefix := range []string{"./", "../", "/", "~/", ".\\", "..\\", "~\\"} {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

func joinPath(base, sub string) string {
	sub = strings.Trim(sub, "/")
	if sub == "" {
		return base
	}
	if base == "" {
		return sub
	}
	return base + "/" + sub
}

// gitPURL builds pkg:git/host/repo[@version][#subpath].
func gitPURL(repo, version, subpath string) string {
	i := strings.LastIndexByte(repo, '/')
	p := purl.New("git", repo[:i], repo[i+1:], version, nil)
	p.Subpath = subpath
	return p.String()
}

type lockParser struct{}

type lockfile struct {
	LockfileVersion yaml.Node   `yaml:"lockfile_version"`
	Replacements    yaml.Node   `yaml:"replacements"`
	Dependencies    []lockEntry `yaml:"dependencies"`
}

type lockEntry struct {
	RepoURL            string            `yaml:"repo_url"`
	Host               string            `yaml:"host"`
	Port               int               `yaml:"port"`
	ResolvedRef        string            `yaml:"resolved_ref"`
	ResolvedCommit     string            `yaml:"resolved_commit"`
	Version            string            `yaml:"version"`
	VirtualPath        string            `yaml:"virtual_path"`
	Depth              *int              `yaml:"depth"`
	Source             string            `yaml:"source"`
	ResolvedURL        string            `yaml:"resolved_url"`
	ResolvedHash       string            `yaml:"resolved_hash"`
	LocalPath          string            `yaml:"local_path"`
	ContentHash        string            `yaml:"content_hash"`
	IsDev              bool              `yaml:"is_dev"`
	DeployedFileHashes map[string]string `yaml:"deployed_file_hashes"`
}

// supportedLockVersions omits the unratified source-replacement version "3".
var supportedLockVersions = []string{"1", "2"}

func (p *lockParser) Parse(filename string, content []byte) (*core.Result, error) {
	var lock lockfile
	if err := yaml.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
	if err := checkLockVersion(lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for i, entry := range lock.Dependencies {
		if entry.RepoURL == "" {
			return nil, &core.ParseError{Filename: filename, Err: fmt.Errorf("dependency %d has no repo_url", i)}
		}
		depth := 1
		if entry.Depth != nil {
			depth = *entry.Depth
		}
		if entry.RepoURL == selfRepoURL || depth == 0 {
			continue
		}
		dep := lockDependency(entry)
		dep.Direct = depth == 1
		dep.Scope = core.Runtime
		if entry.IsDev {
			dep.Scope = core.Development
		}
		dep.Artifacts = lockArtifacts(entry)
		deps = append(deps, dep)
	}
	return &core.Result{Dependencies: deps}, nil
}

func checkLockVersion(lock lockfile) error {
	if lock.Replacements.Kind != 0 {
		return fmt.Errorf("source replacement lockfiles are not supported")
	}
	v := lock.LockfileVersion
	if v.Kind == 0 {
		return nil // pre-versioned lockfiles are read as version 1
	}
	if v.Kind != yaml.ScalarNode || v.Tag != "!!str" || !slices.Contains(supportedLockVersions, v.Value) {
		return fmt.Errorf("unsupported lockfile_version %q", v.Value)
	}
	return nil
}

func lockDependency(entry lockEntry) core.Dependency {
	switch entry.Source {
	case "local":
		return localDependency(cmp.Or(entry.LocalPath, entry.RepoURL), entry.Version)
	case "registry":
		dep := registryDependency(entry.RepoURL, entry.VirtualPath, entry.Version, "")
		dep.Source.Value = entry.ResolvedURL
		dep.Integrity = integrity(entry.ResolvedHash)
		return dep
	}

	dep := core.Dependency{
		Version: cmp.Or(entry.ResolvedCommit, entry.Version),
		Source:  core.Source{Kind: core.SourceGit, Value: entry.RepoURL, Ref: entry.ResolvedRef},
	}
	repo := lockRepo(entry)
	if repo == "" {
		dep.Name = joinPath(entry.RepoURL, entry.VirtualPath)
		dep.NoPURL = true
		return dep
	}
	dep.Name = joinPath(repo, entry.VirtualPath)
	dep.Ecosystem = "git"
	dep.PURL = gitPURL(repo, dep.Version, strings.Trim(entry.VirtualPath, "/"))
	return dep
}

// lockRepo returns host/repository for a Git lock entry.
func lockRepo(entry lockEntry) string {
	host, path := splitLocator(entry.RepoURL)
	if entry.Host != "" && !isURL(entry.RepoURL) && !isHostQualified(entry.RepoURL) {
		host, path = strings.ToLower(entry.Host), entry.RepoURL
	}
	if host == "" {
		return ""
	}
	if entry.Port != 0 && !strings.Contains(host, ":") {
		host = fmt.Sprintf("%s:%d", host, entry.Port)
	}
	segments, _ := repoSegments(host, path)
	if len(segments) < 2 {
		return ""
	}
	return host + "/" + repoPathCase(host, strings.Join(segments, "/"))
}

// lockArtifacts keeps archive, package-tree and deployed-file hashes distinct.
func lockArtifacts(entry lockEntry) []core.ArtifactIntegrity {
	var artifacts []core.ArtifactIntegrity
	if entry.ResolvedHash != "" {
		artifacts = append(artifacts, artifact(artifactArchive, entry.ResolvedHash))
	}
	if entry.ContentHash != "" {
		artifacts = append(artifacts, artifact(artifactTree, entry.ContentHash))
	}
	for path, hash := range entry.DeployedFileHashes {
		if hash != "" {
			artifacts = append(artifacts, artifact(artifactDeployed+path, hash))
		}
	}
	return artifacts
}

// artifact splits "sha256:<hex>"; bare digests are SHA-256 per the lockfile spec.
func artifact(role, value string) core.ArtifactIntegrity {
	algorithm, digest, ok := strings.Cut(value, ":")
	if !ok {
		algorithm, digest = "sha256", value
	}
	return core.ArtifactIntegrity{Artifact: role, Algorithm: algorithm, Digest: digest}
}

// integrity formats an archive hash in the library's algorithm-digest form.
func integrity(value string) string {
	if value == "" {
		return ""
	}
	a := artifact(artifactArchive, value)
	return a.Algorithm + "-" + a.Digest
}
