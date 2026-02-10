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
	"github.com/git-pkgs/manifests/internal/core"
	"github.com/git-pkgs/purl"
)

// Re-export types from internal/core for public API.
type (
	Kind       = core.Kind
	Scope      = core.Scope
	Dependency = core.Dependency
)

// Re-export constants.
const (
	Manifest   Kind = core.Manifest
	Lockfile   Kind = core.Lockfile
	Supplement Kind = core.Supplement

	Runtime     Scope = core.Runtime
	Development Scope = core.Development
	Test        Scope = core.Test
	Build       Scope = core.Build
	Optional    Scope = core.Optional
)

// ParseResult contains the parsed dependencies from a manifest or lockfile.
type ParseResult struct {
	Ecosystem    string
	Kind         Kind
	Dependencies []Dependency
}

// Parse parses a manifest or lockfile and returns its dependencies.
func Parse(filename string, content []byte) (*ParseResult, error) {
	parser, eco, kind := core.IdentifyParser(filename)
	if parser == nil {
		return nil, &UnknownFileError{Filename: filename}
	}

	deps, err := parser.Parse(filename, content)
	if err != nil {
		return nil, err
	}

	// Generate PURLs for all dependencies
	for i := range deps {
		version := ""
		if kind == Lockfile || kind == Supplement {
			version = deps[i].Version
		}
		deps[i].PURL = makePURL(eco, deps[i].Name, version, deps[i].RegistryURL)
	}

	return &ParseResult{
		Ecosystem:    eco,
		Kind:         kind,
		Dependencies: deps,
	}, nil
}

// makePURL creates a Package URL for a dependency.
func makePURL(ecosystem, name, version, registryURL string) string {
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
