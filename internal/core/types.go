// Package core provides shared types and the parser registry.
package core

// Kind distinguishes manifest files from lockfiles.
type Kind string

const (
	Manifest Kind = "manifest"
	Lockfile Kind = "lockfile"
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

// Dependency represents a parsed dependency from a manifest or lockfile.
type Dependency struct {
	Name        string
	Version     string
	Scope       Scope
	Integrity   string
	Direct      bool
	PURL        string
	RegistryURL string
}

// Parser is the interface implemented by all manifest parsers.
type Parser interface {
	Parse(filename string, content []byte) ([]Dependency, error)
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
