package manifests

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// RepositoryReader provides bounded access to files in a repository. Paths and
// glob patterns use forward slashes and are relative to the repository root.
// Glob must support the doublestar dialect, including recursive ** patterns.
// ReadFile should return an error matching fs.ErrNotExist for absent files.
type RepositoryReader interface {
	ReadFile(name string) ([]byte, error)
	Glob(pattern string) ([]string, error)
}

// FSReader adapts an fs.FS to RepositoryReader. It supports recursive ** globs.
type FSReader struct {
	fsys fs.FS
}

// NewFSReader returns a repository reader rooted at fsys. Use os.DirFS to
// discover manifests in a working tree.
func NewFSReader(fsys fs.FS) *FSReader {
	return &FSReader{fsys: fsys}
}

// ReadFile reads a repository-relative file.
func (r *FSReader) ReadFile(name string) ([]byte, error) {
	if r == nil || r.fsys == nil {
		return nil, errors.New("nil repository filesystem")
	}
	return fs.ReadFile(r.fsys, name)
}

// Glob returns files matching a repository-relative doublestar pattern.
func (r *FSReader) Glob(pattern string) ([]string, error) {
	if r == nil || r.fsys == nil {
		return nil, errors.New("nil repository filesystem")
	}
	return doublestar.Glob(r.fsys, pattern,
		doublestar.WithFilesOnly(),
		doublestar.WithFailOnIOErrors(),
	)
}

// DiscoveredManifest identifies a project or workspace manifest. ParentPath is
// the repository-relative workspace configuration that selected a nested
// manifest; it is empty for root and path-based manifests.
type DiscoveredManifest struct {
	Path       string
	Ecosystem  string
	Kind       Kind
	ParentPath string
}

type manifestDiscovery struct {
	reader   RepositoryReader
	items    map[discoveredManifestKey]DiscoveredManifest
	warnings []error
}

func (d *manifestDiscovery) warn(err error) {
	d.warnings = append(d.warnings, err)
}

type discoveredManifestKey struct {
	path      string
	ecosystem string
	kind      Kind
}

// DiscoverManifests returns the project and workspace manifests selected from
// a rooted repository. It does not parse dependency declarations. Warnings
// report workspace configurations or lookups that could not be processed;
// successfully discovered manifests are still returned.
func DiscoverManifests(reader RepositoryReader) ([]DiscoveredManifest, []error) {
	if reader == nil {
		return nil, []error{errors.New("nil repository reader")}
	}

	discovery := &manifestDiscovery{
		reader: reader,
		items:  make(map[discoveredManifestKey]DiscoveredManifest),
	}
	for _, pattern := range []string{"*", ".github/workflows/*.yml", ".github/workflows/*.yaml"} {
		if err := discovery.addMatches(pattern, ""); err != nil {
			discovery.warn(fmt.Errorf("discovering manifests matching %q: %w", pattern, err))
		}
	}

	workspaceDiscoveries := []func() error{
		discovery.discoverCargoWorkspace,
		discovery.discoverGoWorkspace,
		discovery.discoverNPMWorkspace,
		discovery.discoverPnpmWorkspace,
	}
	for _, discover := range workspaceDiscoveries {
		if err := discover(); err != nil {
			discovery.warn(err)
		}
	}

	return discovery.sorted(), discovery.warnings
}

func (d *manifestDiscovery) addMatches(pattern, parentPath string) error {
	matches, err := d.reader.Glob(pattern)
	if err != nil {
		return err
	}
	sort.Strings(matches)
	for _, match := range matches {
		d.add(match, parentPath)
	}
	return nil
}

func (d *manifestDiscovery) add(manifestPath, parentPath string) {
	manifestPath, ok := normalizeRepositoryPath(manifestPath)
	if !ok {
		return
	}
	matches := IdentifyAll(manifestPath)
	if len(matches) == 0 {
		return
	}
	if parentPath != "" {
		var valid bool
		parentPath, valid = normalizeRepositoryPath(parentPath)
		if !valid {
			return
		}
	}

	for _, match := range matches {
		key := discoveredManifestKey{path: manifestPath, ecosystem: match.Ecosystem, kind: match.Kind}
		item := DiscoveredManifest{
			Path:       manifestPath,
			Ecosystem:  match.Ecosystem,
			Kind:       match.Kind,
			ParentPath: parentPath,
		}
		if current, exists := d.items[key]; !exists || current.ParentPath == "" && parentPath != "" {
			d.items[key] = item
		}
	}
}

func (d *manifestDiscovery) addWorkspaceManifests(
	includePatterns, excludePatterns []string,
	manifestName, parentPath string,
) error {
	excluded, err := d.workspaceManifestPaths(excludePatterns, manifestName)
	if err != nil {
		return err
	}

	included := make(map[string]struct{})
	for _, pattern := range includePatterns {
		normalized, ok := normalizeRepositoryPattern(pattern)
		if !ok {
			continue
		}
		matches, err := d.reader.Glob(path.Join(normalized, manifestName))
		if err != nil {
			return fmt.Errorf("expanding workspace pattern %q: %w", pattern, err)
		}
		if len(matches) == 0 && isLiteralPattern(normalized) {
			d.warn(fmt.Errorf("%s: workspace member %q has no %s", parentPath, pattern, manifestName))
		}
		for _, match := range matches {
			if p, valid := normalizeRepositoryPath(match); valid {
				included[p] = struct{}{}
			}
		}
	}

	paths := make([]string, 0, len(included))
	for manifestPath := range included {
		if _, skip := excluded[manifestPath]; !skip {
			paths = append(paths, manifestPath)
		}
	}
	sort.Strings(paths)
	for _, manifestPath := range paths {
		d.add(manifestPath, parentPath)
	}
	return nil
}

// isLiteralPattern reports whether a workspace member pattern names a
// single directory rather than a glob. Wildcard entries that expand to
// nothing are not treated as missing.
func isLiteralPattern(pattern string) bool {
	return !strings.ContainsAny(pattern, "*?[{")
}

func (d *manifestDiscovery) workspaceManifestPaths(patterns []string, manifestName string) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	for _, pattern := range patterns {
		pattern, ok := normalizeRepositoryPattern(pattern)
		if !ok {
			continue
		}
		matches, err := d.reader.Glob(path.Join(pattern, manifestName))
		if err != nil {
			return nil, fmt.Errorf("expanding workspace pattern %q: %w", pattern, err)
		}
		for _, match := range matches {
			if normalized, valid := normalizeRepositoryPath(match); valid {
				result[normalized] = struct{}{}
			}
		}
	}
	return result, nil
}

func (d *manifestDiscovery) readOptional(name string) ([]byte, bool, error) {
	content, err := d.reader.ReadFile(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return content, true, nil
}

func (d *manifestDiscovery) sorted() []DiscoveredManifest {
	result := make([]DiscoveredManifest, 0, len(d.items))
	for _, item := range d.items {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Path != result[j].Path {
			return result[i].Path < result[j].Path
		}
		if result[i].Ecosystem != result[j].Ecosystem {
			return result[i].Ecosystem < result[j].Ecosystem
		}
		return result[i].Kind < result[j].Kind
	})
	return result
}

func normalizeRepositoryPath(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\`, "/"))
	if value == "" || strings.HasPrefix(value, "/") {
		return "", false
	}
	value = path.Clean(value)
	if value == "." || value == ".." || strings.HasPrefix(value, "../") {
		return "", false
	}
	return value, true
}

func normalizeRepositoryPattern(value string) (string, bool) {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\`, "/"))
	value = strings.TrimSuffix(value, "/")
	if value == "." {
		return value, true
	}
	return normalizeRepositoryPath(value)
}
