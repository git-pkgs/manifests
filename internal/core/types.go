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
	SourceURL      SourceKind = "url"
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

// Publication preserves package publication settings declared by a manifest.
type Publication struct {
	// Provenance reports an explicit request to publish provenance. Nil means
	// the manifest does not declare the setting.
	Provenance *bool
}

// Metadata holds descriptive package fields as written in the manifest. A nil
// pointer or slice means the field is absent; a pointer to "" or an empty
// slice means it is declared empty.
type Metadata struct {
	// Summary is a short one-line description, when the format keeps it
	// separate from Description.
	Summary     *string
	Description *string
	// DescriptionContentType is the declared media type of Description,
	// such as "text/markdown".
	DescriptionContentType *string
	Keywords               []string
	Homepage               *string
	// Links holds other declared URLs, labelled with the format's own term.
	Links []Link
	// Readme is a declared README path. It is a reference and is not read.
	Readme *string
}

// Link is a declared URL with the label the manifest gives it.
type Link struct {
	Label string
	URL   string
}

// Dependency represents a parsed dependency from a manifest or lockfile.
type Dependency struct {
	Name    string
	Version string
	Scope   Scope
	// Integrity is an opaque verification value derived from the manifest or
	// lockfile. Its digest encoding is ecosystem-specific.
	Integrity string
	// Artifacts holds verification values for individual files of this
	// package version, such as a package archive and its manifest, when the
	// source records them separately. It is optional and independent of
	// Integrity. Parse orders the records and drops exact duplicates.
	Artifacts []ArtifactIntegrity `json:",omitempty"`
	Direct    bool
	// PURL identifies the package. Parse preserves a parser-supplied value and
	// otherwise builds it from Ecosystem, Name and, for lockfiles and
	// supplements, Version.
	PURL string
	// Ecosystem is the dependency's ecosystem name, in the same form as
	// ParseResult.Ecosystem. A parser sets it when the package belongs to a
	// different ecosystem than the file, such as an npm package imported from
	// deno.json. Parse fills in the file's ecosystem otherwise and leaves it
	// empty when NoPURL is set.
	Ecosystem string
	// NoPURL marks a source-only entry, such as a direct download URL, whose
	// package identity is unknown. Parse leaves its PURL empty.
	NoPURL      bool
	RegistryURL string
	// Source is set only when this dependency has an explicit source override.
	// It is intentionally separate from RegistryURL because paths and Git
	// repositories are not package registries.
	Source Source
}

// ArtifactIntegrity is a verification value for one file of a dependency's
// package version.
type ArtifactIntegrity struct {
	// Artifact names the file the value verifies, using the source format's
	// own term, such as "toml" or "fpkg" in a Flix packages.lock.
	Artifact string
	// Algorithm is the hash algorithm as named by the source, such as
	// "sha256". It is empty when the source does not name one.
	Algorithm string
	// Digest is the hash value in the source's encoding, without the
	// algorithm name.
	Digest string
}

// Replacement records a directive that substitutes the package or source used
// for another package, without applying it. Location is ecosystem-specific and
// should be treated as an opaque identity.
type Replacement struct {
	Name               string
	Version            string
	ReplacementName    string
	ReplacementVersion string
	Source             Source
	Location           string
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
	PURL string
	// Ecosystem is the declaration's ecosystem name, in the same form as
	// ParseResult.Ecosystem. A parser sets it when the package belongs to a
	// different ecosystem than the file. Parse fills in the file's ecosystem
	// otherwise and leaves it empty when NoPURL is set.
	Ecosystem string
	// NoPURL marks a source-only entry whose package identity is unknown.
	// Parse leaves its PURL empty.
	NoPURL   bool
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
	// Replacements holds replace or override directives in source order.
	Replacements []Replacement
	// Sources preserves manifest-level source declarations in source order.
	// These entries are configuration, not evidence that any dependency was
	// resolved from a particular source.
	Sources []Source
	// Scripts maps declared hook or task names to ordered commands or script paths.
	Scripts map[string][]string
	// Publication holds settings that affect publishing this package.
	Publication Publication
	// Metadata holds descriptive package fields.
	Metadata Metadata
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
