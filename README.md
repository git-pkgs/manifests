# manifests

A Go library for parsing package manager manifest and lockfiles. Extracts dependencies with version constraints, scopes, and integrity hashes.

## Installation

```bash
go get github.com/git-pkgs/manifests
```

## Usage

```go
package main

import (
    "fmt"
    "os"
    "github.com/git-pkgs/manifests"
)

func main() {
    content, _ := os.ReadFile("package.json")
    result, err := manifests.Parse("package.json", content)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Ecosystem: %s\n", result.Ecosystem)
    fmt.Printf("Kind: %s\n", result.Kind)
    fmt.Printf("Package: %s %s\n", result.Name, result.Version)
    for _, dep := range result.Dependencies {
        fmt.Printf("  %s@%s (%s)\n", dep.Name, dep.Version, dep.Scope)
    }
}
```

## Supported Ecosystems

| Ecosystem | Manifests | Lockfiles |
|-----------|-----------|-----------|
| alpine | APKBUILD | |
| arch | PKGBUILD | |
| asdf | .tool-versions | |
| bazel | MODULE.bazel | |
| bower | bower.json | |
| brew | Brewfile | Brewfile.lock.json |
| cargo | Cargo.toml | Cargo.lock |
| carthage | Cartfile, Cartfile.private | Cartfile.resolved |
| chef | metadata.rb, metadata.json, Berksfile | |
| clojars | project.clj | |
| cocoapods | Podfile, *.podspec | Podfile.lock |
| composer | composer.json | composer.lock |
| conan | conanfile.txt, conanfile.py | conan.lock |
| conda | environment.yml, environment.yaml | conda-lock.yml |
| cpan | cpanfile, Makefile.PL, Build.PL, dist.ini, META.json, META.yml | cpanfile.snapshot |
| cran | DESCRIPTION | renv.lock |
| crystal | shard.yml | shard.lock |
| deno | deno.json, deno.jsonc | deno.lock |
| docker | Dockerfile, Dockerfile.*, docker-compose.yml, docker-compose.yaml, compose.yml, compose.yaml | |
| dub | dub.json, dub.sdl | |
| elm | elm.json, elm-package.json | |
| gem | Gemfile, gems.rb, *.gemspec | Gemfile.lock, gems.locked |
| git | .gitmodules | |
| github-actions | .github/workflows/*.yml, .github/workflows/*.yaml | .github/workflows/actions.lock |
| golang | go.mod, Godeps, glide.yaml, Gopkg.toml | Godeps.json, glide.lock, Gopkg.lock, vendor.json, go-resolved-dependencies.json, vendor/manifest, go.graph |
| guix | manifest.scm, *-manifest.scm | |
| hackage | *.cabal | stack.yaml.lock, cabal.config, cabal.project.freeze |
| haxelib | haxelib.json | |
| helm | Chart.yaml, requirements.yaml | Chart.lock, requirements.lock |
| hex | mix.exs, gleam.toml | mix.lock, rebar.lock |
| ips | *.p5m | |
| julia | Project.toml, REQUIRE | Manifest.toml |
| lean | lakefile.toml, lakefile.lean | lake-manifest.json |
| luarocks | *.rockspec | |
| maven | pom.xml, ivy.xml, build.gradle, build.gradle.kts, build.sbt | gradle.lockfile, gradle-dependencies-q.txt, maven-resolved-dependencies.txt, maven.graph.json, verification-metadata.xml, dependencies.lock, gradle-html-dependency-report.js, dependencies-*.dot, *-compile.xml, *-test.xml, *-runtime.xml, *-provided.xml |
| nimble | *.nimble | |
| nix | flake.nix | flake.lock, sources.json |
| pre-commit | .pre-commit-config.yaml, prek.toml | |
| npm | package.json | package-lock.json, npm-shrinkwrap.json, yarn.lock, pnpm-lock.yaml, bun.lock, npm-ls.json |
| nuget | *.csproj, *.vbproj, *.fsproj, *.nuspec, packages.config, Directory.Packages.props, Project.json | packages.lock.json, paket.lock, project.assets.json, *.deps.json, Project.lock.json |
| opam | opam, *.opam | |
| pub | pubspec.yaml | pubspec.lock |
| pypi | requirements.txt, Pipfile, pyproject.toml, setup.py, setup.cfg | Pipfile.lock, poetry.lock, pdm.lock, uv.lock, pip-dependency-graph.json, pip-resolved-dependencies.txt, pylock.toml |
| rpm | *.spec | |
| swift | Package.swift | Package.resolved |
| vcpkg | vcpkg.json | |

## Lockfile Feature Support

| Lockfile | Registry URL | Integrity | Scope | Direct |
|----------|:------------:|:---------:|:-----:|:------:|
| package-lock.json | ✓ | ✓ | ✓ | ✓ |
| npm-shrinkwrap.json | ✓ | ✓ | ✓ | ✓ |
| yarn.lock | ✓ | ✓ | | |
| pnpm-lock.yaml | ✓ | ✓ | ✓ | |
| bun.lock | ✓ | ✓ | | |
| npm-ls.json | ✓ | ✓ | ✓ | |
| deno.lock | | ✓ | | |
| .github/workflows/actions.lock | | ✓ | | ✓ |
| Gemfile.lock | ✓ | ✓ | | ✓ |
| gems.locked | ✓ | ✓ | | ✓ |
| Cargo.lock | ✓ | ✓ | | |
| Cartfile.resolved | | | | |
| Chart.lock | ✓ | | | ✓ |
| requirements.lock | ✓ | | | ✓ |
| poetry.lock | ✓ | ✓ | ✓ | |
| Pipfile.lock | ✓ | ✓ | ✓ | |
| pdm.lock | | ✓ | ✓ | |
| uv.lock | ✓ | ✓ | | |
| pylock.toml | | ✓ | | |
| pip-resolved-dependencies.txt | | | | |
| pip-dependency-graph.json | | | | |
| composer.lock | ✓ | ✓ | ✓ | |
| Podfile.lock | | ✓ | | ✓ |
| mix.lock | | ✓ | | |
| rebar.lock | | ✓ | | |
| pubspec.lock | ✓ | ✓ | | ✓ |
| conan.lock | | | ✓ | |
| conda-lock.yml | ✓ | ✓ | ✓ | |
| cpanfile.snapshot | | | | |
| packages.lock.json | | ✓ | | ✓ |
| paket.lock | | | | |
| project.assets.json | | ✓ | | |
| *.deps.json | | ✓ | | |
| Project.lock.json | | ✓ | | |
| stack.yaml.lock | | ✓ | | |
| cabal.config | | | | |
| cabal.project.freeze | | | | |
| Godeps.json | | | | |
| glide.lock | | | | |
| Gopkg.lock | | | | |
| vendor.json | | | | |
| go-resolved-dependencies.json | | | ✓ | ✓ |
| vendor/manifest | | | | |
| go.graph | | | | ✓ |
| Manifest.toml | | | | |
| gradle.lockfile | | | ✓ | |
| gradle-dependencies-q.txt | | | ✓ | |
| maven-resolved-dependencies.txt | | | ✓ | |
| maven.graph.json | | | ✓ | |
| verification-metadata.xml | | | | |
| dependencies.lock | | | ✓ | ✓ |
| gradle-html-dependency-report.js | | | ✓ | |
| dependencies-*.dot | | | ✓ | |
| *-compile.xml, *-test.xml, *-runtime.xml, *-provided.xml | | | ✓ | |
| renv.lock | | ✓ | | |
| shard.lock | | | | |
| flake.lock | | ✓ | | |
| sources.json | | ✓ | | |
| Brewfile.lock.json | | ✓ | | ✓ |
| lake-manifest.json | ✓ | | | ✓ |
| Package.resolved | | | | |

**Supplement files:** go.sum is parsed as a supplement rather than a lockfile. It provides integrity hashes that can be matched against go.mod dependencies by name and version, but it doesn't represent a standalone dependency tree.

## API

### Parse

Parses a manifest or lockfile and returns extracted dependencies.

```go
func Parse(filename string, content []byte) (*ParseResult, error)
```

### Identify

Returns the ecosystem and kind for a filename without parsing.

```go
func Identify(filename string) (ecosystem string, kind Kind, ok bool)
```

### IdentifyAll

Returns all matching ecosystems for a filename (some files match multiple parsers).

```go
func IdentifyAll(filename string) []Match
```

### Ecosystems

Returns a list of supported ecosystems.

```go
func Ecosystems() []string
```

### DiscoverManifests

Discovers root manifests, GitHub Actions workflows, and declared Cargo, Go,
npm/Yarn, and pnpm workspace members. Discovery is repository-aware while
`Parse` remains a pure single-file operation.

```go
reader := manifests.NewFSReader(os.DirFS("."))
found, warnings := manifests.DiscoverManifests(reader)
```

Callers reading historical revisions can implement `RepositoryReader` over a
Git tree. Paths and glob patterns are rooted, slash-separated repository paths.
Workspace records set `ParentPath` to the configuration that selected them.
Warnings report malformed workspace configuration or failed lookups without
discarding manifests that were discovered successfully.

### DiscoverVendors

Discovers package-manager vendor roots and exact package identities stored in
the repository. The initial implementation recognizes npm `node_modules`
trees exposed by the supplied reader, Go `vendor/modules.txt`, Python
`[tool.vendoring]` configuration, and Cargo directory sources selected through
`.cargo/config.toml` or `.cargo/config`.

```go
reader := manifests.NewFSReader(os.DirFS("."))
found, warnings := manifests.DiscoverVendors(reader)
```

`VendorDiscovery.Roots` classifies each vendor directory by ecosystem and
records the configuration or inventory that selected it. Each
`VendoredDependency` has `Kind == Vendor`, an exact package identity and PURL,
its vendor root, and the evidence file from which the identity was read.
Results are normalized and deterministic. Invalid configuration, unreadable
evidence, and incomplete package identities are returned as warnings without
discarding successful discoveries.

## Types

This module is pre-v1 and exported result structs may gain additive metadata
fields in minor releases. Use keyed composite literals when constructing
`Dependency`, `Declaration`, or `ParseResult` values.

### Dependency

```go
type Dependency struct {
    Name        string // Package name
    Version     string // Version constraint or resolved version
    Scope       Scope  // runtime, development, test, build, optional
    Integrity   string // Opaque verification value, when available
    Direct      bool   // True if declared directly, false if transitive
    PURL        string // Package URL (pkg:ecosystem/name@version)
    RegistryURL string // Source registry URL (if non-default)
    Source      Source // Explicit Git, path, or ecosystem source override
}
```

`Integrity` is an opaque verification value derived from the source file. The
digest encoding remains ecosystem-specific, and parsers may add an algorithm
prefix such as `sha256-` when the source stores it separately. That prefix does
not imply Subresource Integrity: the digest may use hexadecimal, base64, Nix
base32, or another ecosystem-specific form. Consumers should only decode or
convert the value with knowledge of the source format.

When a dependency comes from a non-default registry, the PURL includes a `repository_url` qualifier (e.g., `pkg:npm/foo@1.0.0?repository_url=https://npm.mycompany.com/`). Default registries like registry.npmjs.org, pypi.org, and rubygems.org are not included in the PURL.

### Declaration

```go
type Declaration struct {
    Name     string // Package name
    Version  string // Version requirement as written in the manifest
    Scope    Scope  // runtime, development, test, build, optional
    Direct   bool   // Direct rather than generated or transitive
    PURL     string // Versionless Package URL
    Location string // Opaque parser-defined identity within the manifest
    Source   Source // Explicit source override as written
}
```

Declarations preserve source-level references without applying inheritance,
merging, interpolation, or other effective-model resolution. Consumers can use
`Location` to match the same logical entry across edits, but should not parse
its ecosystem-specific value. A declaration PURL omits the version because the
raw requirement may be a range or property expression. When a parser supplies
its own PURL, `Parse` preserves it so one manifest can refer to packages from
different ecosystems. Otherwise `Parse` builds the PURL from the parser's
ecosystem.

`Direct` distinguishes explicit requirements from generated or transitive
entries when the source format records that distinction, such as `go.mod`.

### Source

```go
type Source struct {
    Kind   SourceKind // registry, git, path, or github
    Value  string     // Literal URL, path, or ecosystem coordinate
    Branch string     // Literal branch selector
    Tag    string     // Literal tag selector
    Ref    string     // Literal revision selector
    Rel    string     // Literal repository-relative cookbook path
}
```

`Source` records source syntax without claiming that dependency resolution
used that location. Manifest-level source configuration is preserved in
`ParseResult.Sources` in declaration order. An explicit dependency override is
stored on both its `Dependency` and `Declaration`; dependencies with no
override have an empty `Source`. `RegistryURL` remains reserved for resolved or
otherwise attributable package registries and is not used for Git repositories
or local paths. Chef Git and GitHub sources also retain literal `branch`, `tag`,
`ref`, and `rel` options without resolving them.

Parsers that do not preserve source locations leave `Declarations` empty.
Declarations are available for `package.json`, Cargo manifests, `go.mod`, Python
requirements files, `pyproject.toml`, GitHub Actions workflows, `gleam.toml`,
`pom.xml`, NuGet project and package files, and
`Directory.Packages.props`. The Maven parser includes parents,
dependencies, dependency management, plugins, plugin dependencies, plugin
management, build extensions, and their profile-scoped forms.

### ParseResult

```go
type ParseResult struct {
    Ecosystem    string       // npm, gem, pypi, golang, cargo, etc.
    Kind         Kind         // manifest, lockfile, or supplement
    Name         string       // the package's own name, when the format declares one
    Version      string       // the package's own version, when declared
    Licenses     []string     // raw declared license values
    LicenseFile  string       // manifest-relative path to a declared license file
    Digest       string       // file-level verification value, when present
    Dependencies []Dependency
    Declarations []Declaration
    Sources      []Source      // Ordered manifest-level source declarations
}
```

`Name` and `Version` are populated for manifest formats that declare their own package identity (Cargo.toml `[package]`, package.json `"name"`, go.mod `module`, `.gemspec`, and so on). They are empty for lockfiles and for dependency-only files like Gemfile or requirements.txt.

`Licenses` contains decoded values as declared by the manifest; it does not normalize them into SPDX expressions. `LicenseFile` is populated when a format explicitly identifies a license file. Both are empty for formats without license metadata.

`Digest` contains a file-level verification value when the format defines one. For `Chart.lock`, it covers the dependency declarations from `Chart.yaml` and is separate from each dependency's `Integrity` value.

Chef cookbook PURLs remain empty while `chef` is only a candidate Package URL
type without accepted name and namespace rules.

### Vendor Discovery

```go
type VendorDiscovery struct {
    Roots        []VendorRoot
    Dependencies []VendoredDependency
}

type VendorRoot struct {
    Path       string
    Ecosystem  string
    ConfigPath string
}

type VendoredDependency struct {
    Name         string
    Version      string
    Ecosystem    string
    Kind         Kind
    PURL         string
    RootPath     string
    EvidencePath string
}
```

### Kind

```go
const (
    Manifest   Kind = "manifest"   // Declared dependencies with version constraints
    Lockfile   Kind = "lockfile"   // Resolved dependencies with exact versions
    Supplement Kind = "supplement" // Provides extra data (e.g. integrity hashes) for a manifest's dependencies
    Vendor     Kind = "vendor"     // Exact package identity stored under a vendor root
)
```

### Scope

```go
const (
    Runtime     Scope = "runtime"
    Development Scope = "development"
    Test        Scope = "test"
    Build       Scope = "build"
    Optional    Scope = "optional"
)
```
