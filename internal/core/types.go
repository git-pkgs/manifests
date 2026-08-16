// Package core provides shared types and the parser registry.
package core

// Kind distinguishes manifest, lockfile, supplement, and vendor records.
type Kind string

const (
	Manifest   Kind = "manifest"
	Lockfile   Kind = "lockfile"
	Supplement Kind = "supplement"
	Vendor     Kind = "vendor"
)

// Scope indicates when a dependency is required.
type Scope string

const (
	Runtime     Scope = "runtime"
	Development Scope = "development"
	Test        Scope = "test"
	Build       Scope = "build"
	Optional    Scope = "optional"
)

// SourceKind identifies the kind of location named by a source declaration.
// The value describes syntax in the manifest, not a resolved package source.
type SourceKind string

const (
	SourceRegistry SourceKind = "registry"
	SourceGit      SourceKind = "git"
	SourcePath     SourceKind = "path"
	SourceGitHub   SourceKind = "github"
)

// Source preserves an explicit source declaration without claiming that a
// dependency was resolved from it. Value is the literal URL, path, or
// ecosystem-specific coordinate.
type Source struct {
	Kind   SourceKind
	Value  string
	Branch string
	Tag    string
	Ref    string
	Rel    string
}

// Dependency represents a parsed dependency from a manifest or lockfile.
type Dependency struct {
	Name    string
	Version string
	Scope   Scope
	// Integrity is an opaque verification value derived from the manifest or
	// lockfile. Its digest encoding is ecosystem-specific.
	Integrity   string
	Direct      bool
	PURL        string
	RegistryURL string
	// Source is set only when this dependency has an explicit source override.
	// It is intentionally separate from RegistryURL because paths and Git
	// repositories are not package registries.
	Source Source
}

// Declaration is a dependency-like reference at a stable logical location
// in a manifest. Version is the requirement as written in that location,
// before effective-model resolution or inheritance. Location is
// ecosystem-specific and should be treated as an opaque identity.
type Declaration struct {
	Name    string
	Version string
	Scope   Scope
	// Direct reports whether the manifest declares the package as a direct
	// dependency rather than a generated or transitive requirement.
	Direct bool
	// PURL identifies the declared package without a version. Parsers may set
	// it when a file can contain references from more than one ecosystem.
	PURL     string
	Location string
	Source   Source
}

// Result is the output of a single parser.
type Result struct {
	// Name is the package's own name as declared in the manifest, when
	// the format has one. Empty for lockfiles and for manifest formats
	// that only list dependencies (Gemfile, requirements.txt, etc.).
	Name string
	// Version is the package's own version as declared in the manifest.
	Version string
	// Licenses holds the package's declared license values, without
	// normalization.
	Licenses []string
	// LicenseFile is a manifest-relative path to a declared license file.
	LicenseFile string
	// Digest is a file-level verification value whose meaning is defined by
	// the manifest format. It does not apply to individual dependencies.
	Digest       string
	Dependencies []Dependency
	Declarations []Declaration
	// Sources preserves manifest-level source declarations in source order.
	// These entries are configuration, not evidence that any dependency was
	// resolved from a particular source.
	Sources []Source
}

// Parser is the interface implemented by all manifest parsers.
type Parser interface {
	Parse(filename string, content []byte) (*Result, error)
}

// FSRootParser is optionally implemented by parsers that can consult
// neighbouring files on disk (e.g. pom.xml following <relativePath> to a
// parent). The fsRoot argument bounds that lookup; an empty string means
// no filesystem access.
type FSRootParser interface {
	ParseInRoot(filename string, content []byte, fsRoot string) (*Result, error)
}

// ParseError is returned when parsing fails.
type ParseError struct {
	Filename string
	Err      error
}

func (e *ParseError) Error() string {
	return "failed to parse " + e.Filename + ": " + e.Err.Error()
}

func (e *ParseError) Unwrap() error {
	return e.Err
}
