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
	"net/url"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"github.com/package-url/packageurl-go"
)

// defaultRegistryHosts maps ecosystems to their default registry hosts.
// Based on https://github.com/andrew/purl/blob/main/purl-types.json
var defaultRegistryHosts = map[string][]string{
	"npm":        {"registry.npmjs.org", "registry.yarnpkg.com"},
	"pypi":       {"pypi.org", "files.pythonhosted.org"},
	"cargo":      {"crates.io", "index.crates.io", "static.crates.io"},
	"gem":        {"rubygems.org"},
	"composer":   {"packagist.org", "repo.packagist.org"},
	"maven":      {"repo.maven.apache.org", "repo1.maven.org", "central.maven.org"},
	"golang":     {"pkg.go.dev", "proxy.golang.org"},
	"nuget":      {"nuget.org", "api.nuget.org"},
	"cocoapods":  {"cdn.cocoapods.org", "cocoapods.org"},
	"hex":        {"repo.hex.pm", "hex.pm"},
	"pub":        {"pub.dartlang.org", "pub.dev"},
	"hackage":    {"hackage.haskell.org"},
	"cran":       {"cran.r-project.org"},
	"cpan":       {"cpan.org", "metacpan.org"},
	"luarocks":   {"luarocks.org"},
	"conda":      {"repo.anaconda.com", "anaconda.org"},
	"conan":      {"conan.io"},
	"swift":      {"swiftpackageindex.com"},
	"clojars":    {"clojars.org"},
	"elm":        {"package.elm-lang.org"},
	"deno":       {"deno.land"},
	"homebrew":   {"formulae.brew.sh"},
	"docker":     {"hub.docker.com", "docker.io", "registry-1.docker.io"},
	"github":     {"github.com"},
	"bitbucket":  {"bitbucket.org"},
	"huggingface": {"huggingface.co"},
}

// isNonDefaultRegistry returns true if the registryURL is not a default registry for the ecosystem.
func isNonDefaultRegistry(ecosystem, registryURL string) bool {
	if registryURL == "" {
		return false
	}

	defaults, ok := defaultRegistryHosts[ecosystem]
	if !ok {
		return true
	}

	parsed, err := url.Parse(registryURL)
	if err != nil {
		return true
	}

	host := parsed.Host
	if host == "" {
		host = registryURL
	}

	for _, defaultHost := range defaults {
		if host == defaultHost || strings.HasSuffix(host, "."+defaultHost) {
			return false
		}
	}

	return true
}

// Re-export types from internal/core for public API.
type (
	Kind       = core.Kind
	Scope      = core.Scope
	Dependency = core.Dependency
)

// Re-export constants.
const (
	Manifest Kind = core.Manifest
	Lockfile Kind = core.Lockfile

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
		if kind == Lockfile {
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
	purlType := ecosystem
	namespace := ""
	pkgName := name

	switch ecosystem {
	case "npm":
		if strings.HasPrefix(name, "@") {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) == 2 {
				namespace = strings.TrimPrefix(parts[0], "@")
				pkgName = parts[1]
			}
		}
	case "golang":
		if idx := strings.LastIndex(name, "/"); idx > 0 {
			namespace = name[:idx]
			pkgName = name[idx+1:]
		}
	case "maven":
		if strings.Contains(name, ":") {
			parts := strings.SplitN(name, ":", 2)
			namespace = parts[0]
			pkgName = parts[1]
		}
	case "alpine":
		purlType = "apk"
		namespace = "alpine"
	case "arch":
		purlType = "alpm"
		namespace = "arch"
	}

	cleanVersion := version
	if cleanVersion != "" {
		cleanVersion = strings.TrimPrefix(cleanVersion, "^")
		cleanVersion = strings.TrimPrefix(cleanVersion, "~")
		cleanVersion = strings.TrimPrefix(cleanVersion, ">=")
		cleanVersion = strings.TrimPrefix(cleanVersion, "<=")
		cleanVersion = strings.TrimPrefix(cleanVersion, ">")
		cleanVersion = strings.TrimPrefix(cleanVersion, "<")
		cleanVersion = strings.TrimPrefix(cleanVersion, "==")
		cleanVersion = strings.TrimPrefix(cleanVersion, "=")
		cleanVersion = strings.TrimPrefix(cleanVersion, "~>")
		cleanVersion = strings.TrimSpace(cleanVersion)
	}

	var qualifiers packageurl.Qualifiers
	if registryURL != "" && isNonDefaultRegistry(ecosystem, registryURL) {
		qualifiers = packageurl.Qualifiers{{Key: "repository_url", Value: registryURL}}
	}

	purl := packageurl.NewPackageURL(purlType, namespace, pkgName, cleanVersion, qualifiers, "")
	return purl.ToString()
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
