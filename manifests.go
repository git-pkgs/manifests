// Package manifests parses dependency manifest and lockfile formats across package ecosystems.
//
// It supports 40+ ecosystems including npm, gem, pypi, cargo, maven, and more.
// Each ecosystem uses its PURL type as the identifier.
//
// Basic usage:
//
//	result, err := manifests.Parse("package.json", content)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Ecosystem: %s, Kind: %s\n", result.Ecosystem, result.Kind)
//	for _, dep := range result.Dependencies {
//	    fmt.Printf("  %s %s\n", dep.Name, dep.Version)
//	}
package manifests

import (
	"cmp"
	"slices"

	"github.com/git-pkgs/manifests/internal/core"
	"github.com/git-pkgs/purl"
)

// Re-export types from internal/core for public API.
type (
	Kind       = core.Kind
	Scope      = core.Scope
	SourceKind = core.SourceKind
)

// Dependency represents a parsed dependency. Its Integrity field is an opaque
// verification value whose digest encoding depends on the source format.
// Before v1, callers constructing values should use keyed fields so additive
// metadata fields remain source-compatible. Dependency contains a slice, so
// values cannot be compared with == or used as map keys.
type Dependency = core.Dependency

// ArtifactIntegrity is a verification value for one file of a dependency's
// package version, such as its package archive or its manifest.
type ArtifactIntegrity = core.ArtifactIntegrity

// Declaration represents a dependency-like reference at a stable logical
// location in a manifest. Location is ecosystem-specific and opaque.
// Before v1, callers constructing values should use keyed fields so additive
// metadata fields remain source-compatible.
type Declaration = core.Declaration

// Source preserves a literal manifest source declaration. It does not report
// a resolved package location.
type Source = core.Source

// Re-export constants.
const (
	Manifest   Kind = core.Manifest
	Lockfile   Kind = core.Lockfile
	Supplement Kind = core.Supplement
	Vendor     Kind = core.Vendor

	Runtime     Scope = core.Runtime
	Development Scope = core.Development
	Test        Scope = core.Test
	Build       Scope = core.Build
	Optional    Scope = core.Optional

	SourceRegistry SourceKind = core.SourceRegistry
	SourceGit      SourceKind = core.SourceGit
	SourcePath     SourceKind = core.SourcePath
	SourceGitHub   SourceKind = core.SourceGitHub
	SourceURL      SourceKind = core.SourceURL
)

// ParseResult contains the parsed dependencies from a manifest or lockfile.
// Before v1, callers constructing values should use keyed fields so additive
// metadata fields remain source-compatible.
type ParseResult struct {
	Ecosystem string
	Kind      Kind
	// Name is the package's own name as declared in the manifest, when
	// the format has one. Empty for lockfiles and for formats that
	// only list dependencies (Gemfile, requirements.txt, etc.).
	Name string
	// Version is the package's own version as declared in the
	// manifest, when present.
	Version string
	// Licenses holds the package's declared license values, verbatim as
	// written in the manifest. Empty when the format has no license field
	// or none is set. Scalar formats produce a single-element slice.
	Licenses []string
	// LicenseFile is a manifest-relative path to a license file when the
	// format declares one instead of, or as well as, an expression.
	LicenseFile string
	// Digest is a file-level verification value whose meaning is defined by
	// the manifest format. It does not apply to individual dependencies.
	Digest       string
	Dependencies []Dependency
	// Declarations holds source-level references when the parser preserves
	// their logical locations. Unlike Dependencies, these entries are not
	// merged, inherited, or otherwise resolved into an effective model.
	Declarations []Declaration
	// Sources preserves manifest-level source declarations in source order.
	// A source declaration does not imply that any dependency resolved there.
	Sources []Source
	// Scripts maps declared hook or task names to ordered commands, callbacks,
	// or script paths. Keys and values are ecosystem-specific; declarations
	// are not executed or resolved. An empty map does not rule out implicit
	// hooks or code in formats the parser cannot statically inspect.
	Scripts map[string][]string
}

// Options configures Parse.
type Options struct {
	// FSRoot, when non-empty, allows parsers that consult neighbouring
	// files on disk (currently only pom.xml, for parent <relativePath>
	// resolution) to do so within this directory. Paths outside it are
	// refused. When empty, parsing is a pure function of content and no
	// filesystem access occurs; this is the safe choice for untrusted
	// input.
	FSRoot string
}

// Parse parses a manifest or lockfile and returns its dependencies.
func Parse(filename string, content []byte, opts ...Options) (*ParseResult, error) {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	parser, eco, kind := core.IdentifyParser(filename)
	if parser == nil {
		return nil, &UnknownFileError{Filename: filename}
	}

	var res *core.Result
	var err error
	if fp, ok := parser.(core.FSRootParser); ok {
		res, err = fp.ParseInRoot(filename, content, o.FSRoot)
	} else {
		res, err = parser.Parse(filename, content)
	}
	if err != nil {
		return nil, err
	}
	if res == nil {
		res = &core.Result{}
	}

	// Resolve the ecosystem and PURL of every dependency and declaration
	for i := range res.Dependencies {
		dep := &res.Dependencies[i]
		dep.PURL = dependencyPURL(eco, kind, *dep)
		dep.Ecosystem = entryEcosystem(eco, dep.Ecosystem, dep.NoPURL)
		dep.Artifacts = normalizeArtifacts(dep.Artifacts)
	}
	for i := range res.Declarations {
		declaration := &res.Declarations[i]
		declaration.PURL = declarationPURL(eco, *declaration)
		declaration.Ecosystem = entryEcosystem(eco, declaration.Ecosystem, declaration.NoPURL)
	}

	return &ParseResult{
		Ecosystem:    eco,
		Kind:         kind,
		Name:         res.Name,
		Version:      res.Version,
		Licenses:     res.Licenses,
		LicenseFile:  res.LicenseFile,
		Digest:       res.Digest,
		Dependencies: res.Dependencies,
		Declarations: res.Declarations,
		Sources:      res.Sources,
		Scripts:      res.Scripts,
	}, nil
}

// normalizeArtifacts orders artifact integrity records so output does not
// depend on parser iteration order, then drops exact duplicates. An empty
// collection becomes nil.
func normalizeArtifacts(artifacts []core.ArtifactIntegrity) []core.ArtifactIntegrity {
	if len(artifacts) == 0 {
		return nil
	}
	slices.SortFunc(artifacts, func(a, b core.ArtifactIntegrity) int {
		return cmp.Or(
			cmp.Compare(a.Artifact, b.Artifact),
			cmp.Compare(a.Algorithm, b.Algorithm),
			cmp.Compare(a.Digest, b.Digest),
		)
	})
	return slices.Compact(artifacts)
}

// entryEcosystem returns the ecosystem a parser set on a dependency or
// declaration, falling back to the parser's. Entries without a package
// identity have no ecosystem.
func entryEcosystem(ecosystem, entry string, noPURL bool) string {
	if noPURL {
		return ""
	}
	return cmp.Or(entry, ecosystem)
}

// dependencyPURL preserves a parser-supplied package identity or builds one
// from the dependency's ecosystem, falling back to the parser's. Only
// lockfiles and supplements include the version.
func dependencyPURL(ecosystem string, kind Kind, dep core.Dependency) string {
	if dep.NoPURL {
		return ""
	}
	if dep.PURL != "" {
		return dep.PURL
	}
	version := ""
	if kind == Lockfile || kind == Supplement {
		version = dep.Version
	}
	return makePURL(entryEcosystem(ecosystem, dep.Ecosystem, false), dep.Name, version, dep.RegistryURL)
}

// declarationPURL preserves a parser-supplied package identity or builds a
// versionless one from the declaration's ecosystem, falling back to the
// parser's.
func declarationPURL(ecosystem string, declaration core.Declaration) string {
	if declaration.NoPURL {
		return ""
	}
	if declaration.PURL != "" {
		return declaration.PURL
	}
	return makePURL(entryEcosystem(ecosystem, declaration.Ecosystem, false), declaration.Name, "", "")
}

// makePURL creates a Package URL for a dependency.
func makePURL(ecosystem, name, version, registryURL string) string {
	// Chef and Vagrant are still candidate PURL types without accepted name or
	// namespace rules. Keep identities empty instead of inventing mappings that
	// callers could mistake for standardized PURLs.
	if ecosystem == "chef" || ecosystem == "vagrant" {
		return ""
	}
	return purl.BuildPURLString(ecosystem, name, version, registryURL)
}

// Identify returns the ecosystem and kind for a filename without parsing.
func Identify(filename string) (ecosystem string, kind Kind, ok bool) {
	_, eco, k := core.IdentifyParser(filename)
	if eco == "" {
		return "", "", false
	}
	return eco, k, true
}

// Match represents a file type match.
type Match = core.Match

// IdentifyAll returns all matching ecosystems for a filename.
func IdentifyAll(filename string) []Match {
	return core.IdentifyAllParsers(filename)
}

// Ecosystems returns a list of all supported PURL ecosystem types.
func Ecosystems() []string {
	return core.SupportedEcosystems()
}

// UnknownFileError is returned when a file type is not recognized.
type UnknownFileError struct {
	Filename string
}

func (e *UnknownFileError) Error() string {
	return "unknown manifest file: " + e.Filename
}

// ParseError is re-exported from internal/core.
type ParseError = core.ParseError
