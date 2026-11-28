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
| apm | apm.yml | apm.lock.yaml, apm.lock |
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
| golang | go.mod, Godeps, glide.yaml, Gopkg.toml | Godeps.json, glide.lock, Gopkg.lock, vendor.json, go-resolved-dependencies.json, vendor/manifest, go.graph, go.mod.graph, go.modgraph |
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
| pypi | requirements.txt, requirements*.in, requirements/**/*.in, Pipfile, pyproject.toml, setup.py, setup.cfg | Pipfile.lock, poetry.lock, pdm.lock, uv.lock, pip-dependency-graph.json, pip-resolved-dependencies.txt, pylock.toml, pylock.*.toml, pip-inspect.deplock |
| rpm | *.spec | |
| swift | Package.swift, Package@swift-*.swift | Package.resolved |
| vagrant | Vagrantfile | |
| vcpkg | vcpkg.json | |

### Swift output

Swift manifests are read without executing Swift, so variables, conditions,
helper functions and interpolated strings are not evaluated. Packages used by
test targets get `test` scope, and plugin and macro dependencies get `build`.
A package used in several places takes the widest scope, and a regular helper
target used only by tests still counts as `runtime`. If any target reference
doesn't match a declared package or local target, every dependency keeps the
`runtime` default.

`Package.resolved` pins are `Direct: false`, since the file doesn't record which
pins are direct. A pin with no release version uses its revision as `Version`.

### APM output

APM Git dependencies use Ecosystem `git` and a `pkg:git` PURL built from the
repository host and path. A repository subpath, such as one skill in a
monorepo, becomes the PURL subpath, so two subpackages at one commit stay
distinct. The split between repository and subpath follows APM's host rules:
GitHub hosts (`github.com`, `*.ghe.com`) use two repository segments and
lowercase them. Azure DevOps hosts (`dev.azure.com`, `ssh.dev.azure.com`,
`*.visualstudio.com`) drop `_git` and the SSH `v3/` prefix. Hosts that APM
configures through environment variables are not visible offline. Lockfile
Git entries use `resolved_commit` as `Version`, so a commit change is visible
even when the package's display version is unchanged. Installation aliases do
not change identity.

Local paths, registry and marketplace packages, MCP servers and LSP servers
have no PURL. Local paths, registry and marketplace packages keep their
location in `Source`, as do MCP servers from a registry and remote MCP servers
with a `url`. String and object LSP entries and stdio MCP servers carry no
`Source`, since their `command` is an executable rather than a package
location. A registry dependency's `Source.Value` is the registry URL from
`apm.yml`'s `registries`, or the registry name when `apm.yml` does not declare
it. Plain shorthand routes to the default registry only when `apm.yml` declares
`registries.default`; a default set in the user's APM config is not visible
offline. `<self>` is excluded. Lockfile `depth` sets `Direct` and `is_dev` sets
development scope.

Lockfile hashes are kept as `Artifacts` with distinct roles: `archive` for a
registry archive, `tree` for the materialized package tree and
`deployed:<path>` for each deployed file. Only the archive hash is also
reported as `Integrity`. Lockfile versions `"1"` and `"2"` are supported. The
unratified version `"3"` and any `replacements` table are rejected, because a
replacement can install a different source than the declared repository.

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
| pylock.*.toml | | ✓ | | |
| pip-inspect.deplock | | ✓ | | ✓ |
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
| go.mod.graph | | | | ✓ |
| go.modgraph | | | | ✓ |
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
| apm.lock.yaml, apm.lock | | ✓ | ✓ | ✓ |

**Supplement files:** go.sum is parsed as a supplement rather than a lockfile. It provides integrity hashes that can be matched against go.mod dependencies by name and version, but it doesn't represent a standalone dependency tree.

**Config files:** package manager configuration that redirects installs to another registry is parsed with kind `config` and fills `Registries` instead of `Dependencies`. `.npmrc` is a config file too and still reports `Publication.Provenance`.

| File | Ecosystem | Entries |
|------|-----------|---------|
| `.npmrc` | npm | `registry`, `@scope:registry` |
| `.yarnrc.yml` | npm | `npmRegistryServer`, `npmScopes.<scope>.npmRegistryServer` |
| `pip.conf`, `pip.ini` | pypi | `index-url` (replaces), `extra-index-url` (supplements) |
| `.cargo/config.toml`, `.cargo/config` | cargo | `[source.<name>]` `replace-with` chains ending at a `registry` or at a `[registries.<name>]` `index` |
| `.bundle/config` | gem | `BUNDLE_MIRROR__<source>` and `BUNDLE_MIRROR__ALL` |

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
npm/Yarn, and pnpm workspace members. Root config files, including
`.cargo/config.toml`, `.cargo/config` and `.bundle/config`, are reported with
kind `config`. Discovery is repository-aware while
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
    Name        string              // Package name
    Version     string              // Version constraint or resolved version
    Scope       Scope               // runtime, development, test, build, optional
    Integrity   string              // Opaque verification value, when available
    Artifacts   []ArtifactIntegrity // Per-file verification values, when recorded
    Direct      bool                // True if declared directly, false if transitive
    PURL        string              // Package URL (pkg:ecosystem/name@version)
    Ecosystem   string              // Ecosystem name, in the same form as ParseResult.Ecosystem
    NoPURL      bool                // Source-only entry with no known package identity
    RegistryURL string              // Source registry URL (if non-default)
    Source      Source              // Explicit Git, path, or ecosystem source override
    Extras      []string            // Requested optional features, such as Python extras
    Marker      string              // Environment marker, as written and not evaluated
}
```

`Integrity` is an opaque verification value derived from the source file. The
digest encoding remains ecosystem-specific, and parsers may add an algorithm
prefix such as `sha256-` when the source stores it separately. That prefix does
not imply Subresource Integrity: the digest may use hexadecimal, base64, Nix
base32, or another ecosystem-specific form. Consumers should only decode or
convert the value with knowledge of the source format.

`Artifacts` records verification values for individual files of a package
version when the source lists them separately. Flix's `packages.lock`, for
example, can record both a package's `flix.toml` manifest and its `.fpkg`
archive with the same algorithm. Each record keeps its file's role, so a
manifest digest is never mistaken for an archive digest:

```go
type ArtifactIntegrity struct {
    Artifact  string // File role, in the source format's own terms (e.g. "toml", "fpkg")
    Algorithm string // Hash algorithm as named by the source (e.g. "sha256"), if any
    Digest    string // Hash value in the source's encoding, without the algorithm
}
```

One package version stays one `Dependency` however many files it records.
`Parse` sorts each dependency's records by artifact, algorithm and digest, then
drops exact duplicates, so output does not depend on parser iteration order.
`Artifacts` is optional and independent of `Integrity`. Parsers that do not
record per-file values leave it nil, with their `Integrity` values unchanged.

Compatibility: adding `Artifacts` means unkeyed `Dependency` struct literals no
longer compile, so construct values with keyed fields. Because the field is a
slice, `Dependency` values can no longer be compared with `==` or used as map
keys; compare them with `reflect.DeepEqual` or by the fields you need. In JSON
the field is tagged `omitempty`, so encoded dependencies without artifact
records are unchanged.

When a dependency comes from a non-default registry, the PURL includes a `repository_url` qualifier (e.g., `pkg:npm/foo@1.0.0?repository_url=https://npm.mycompany.com/`). Default registries like registry.npmjs.org, pypi.org, and rubygems.org are not included in the PURL.

A file can refer to packages from more than one ecosystem, such as `npm:`
imports in `deno.json`. A parser marks those entries by setting `Ecosystem` to
an ecosystem name in the same form as `ParseResult.Ecosystem`. `Parse` sets
`Ecosystem` on every dependency, using the file's ecosystem when the parser did
not set one, so consumers never need to fall back to `ParseResult.Ecosystem`.
Entries with `NoPURL` keep an empty `Ecosystem` because they have no package
identity.

`Parse` resolves each dependency's PURL in this order:

1. `NoPURL` leaves the PURL empty. Source-only entries, such as direct download
   URLs, keep their location in `Source` without an invented package identity.
2. A PURL supplied by the parser is preserved as written. A parser that
   supplies a PURL from another ecosystem should also set `Ecosystem`.
3. Otherwise the PURL is built from `Ecosystem`. Manifest PURLs omit the
   version. Lockfile and supplement PURLs include it.

`Declaration` follows the same rules and its PURL always omits the version.

### Declaration

```go
type Declaration struct {
    Name        string   // Package name
    Version     string   // Version requirement as written in the manifest
    Scope       Scope    // runtime, development, test, build, optional
    Direct      bool     // Direct rather than generated or transitive
    PURL        string   // Versionless Package URL
    Ecosystem   string   // Ecosystem name, in the same form as ParseResult.Ecosystem
    NoPURL      bool     // Source-only entry with no known package identity
    Location    string   // Opaque parser-defined identity within the manifest
    Source      Source   // Explicit source override as written
    Extras      []string // Requested optional features, such as Python extras
    Marker      string   // Environment marker, as written and not evaluated
    Requirement string   // Original requirement text, when written as one string
}
```

Python requirements written as PEP 508 strings keep their extras and
environment marker. This covers `requirements.txt`, the `[project]`
dependencies, optional dependencies and `build-system.requires` of
`pyproject.toml`, `setup.cfg` and `setup.py`. For the requirement below, `Name` is `requests` and `Version` is
`>=2`, while `Extras` is `["socks"]` and `Marker` is `python_version < '3.12'`:

```text
requests[socks]>=2; python_version < '3.12'
```

The formats that report declarations (`requirements.txt` and `pyproject.toml`)
set the same fields there, with the whole string in `Requirement`. Markers are
not evaluated, so a package listed twice under different markers gives two
entries. A direct reference such as `pip @ https://...` has an empty `Version`.
In `requirements.txt`, per-requirement options such as `--hash` and line
continuations are not part of the requirement.

`pyproject.toml` reports each `build-system.requires` entry with `build` scope
and a declaration under `build-system/requires`. The `build-backend` value is
a module path rather than a package, so it is not recorded.

`requirements*.in` and `requirements/**/*.in` files are pip-tools inputs and
use the `requirements.txt` parser, so `-c` and `-r` lines are skipped. Named
PEP 751 lockfiles such as `pylock.dev.toml` use the `pylock.toml` parser.

`pip-inspect.deplock` is the JSON report of `pip inspect`, as saved by
DepLock. Each installed distribution is reported with runtime scope, and
`Direct` follows the `requested` flag. Entries without a name are skipped.
Packages installed from a direct URL keep it as their `Source`: Git checkouts
use the `git` kind with the commit in `Ref`, local directories use the `path`
kind and other URLs use the `url` kind. An archive's sha256 becomes
`Integrity`.

Declarations preserve source-level references without applying inheritance,
merging, interpolation, or other effective-model resolution. Consumers can use
`Location` to match the same logical entry across edits, but should not parse
its ecosystem-specific value. A declaration PURL omits the version because the
raw requirement may be a range or property expression. When a parser supplies
its own PURL or `Ecosystem`, `Parse` uses it so one manifest can refer to
packages from different ecosystems. Otherwise `Parse` builds the PURL from the
file's ecosystem and records that ecosystem in `Ecosystem`.

`Direct` distinguishes explicit requirements from generated or transitive
entries when the source format records that distinction, such as `go.mod`.

### Source

```go
type Source struct {
    Kind   SourceKind // registry, git, path, github, or url
    Value  string     // Literal URL, path, or ecosystem coordinate
    Branch string     // Literal branch selector
    Tag    string     // Literal tag selector
    Ref    string     // Literal revision selector
    Rel    string     // Literal repository-relative package path
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

Direct artifact or catalog URLs, such as Vagrant `box_url` values, use the
`url` source kind and retain the complete literal URL.

In `deno.json`, remote URL imports use the `url` source kind. Local path and
`file:` imports use the `path` source kind. Both are source-only entries named
by their import map key, with `NoPURL` set. Runtime builtins such as `node:fs`,
inline `data:` modules and bare values are not reported as dependencies.

In `renv.lock`, packages installed from a repository use the `registry` source
kind with the recorded repository name. CRAN packages keep a `cran` PURL, as
do packages from Posit Package Manager recorded as `RSPM`, `PPM` or `P3M`.
Bioconductor packages, including those from repositories such as `BioCsoft`,
use a `bioconductor` PURL. Packages from other repositories keep a `cran` PURL
with a `repository_url` taken from the lockfile's repository list, or from the
repository value when it is a URL. A repository with no known URL has no
package identity. GitHub packages use the `github` source kind with an
`owner/repo` value. GitLab and Bitbucket packages use the `git` source kind
with a `gitlab::owner/repo` or `bitbucket::owner/repo` coordinate. `git`, `URL`
and `Local` packages use the `git`, `url` and `path` source kinds with their
recorded `RemoteUrl`. Git sources keep `RemoteSha`, or `RemoteRef` when no
commit is recorded, in `Ref` and `RemoteSubdir` in `Rel`. These source-only
entries and packages with an unrecognised source have `NoPURL` set. Packages
with no recorded source keep a `cran` PURL.

Parsers that do not preserve source locations leave `Declarations` empty.
Declarations are available for `package.json`, Cargo manifests, `go.mod`, Python
requirements files, `pyproject.toml`, GitHub Actions workflows, `gleam.toml`,
`pom.xml`, NuGet project and package files, `Directory.Packages.props`, Chef
manifests, and `Vagrantfile`. The Maven parser includes parents, dependencies,
dependency management, plugins, plugin dependencies, plugin management, build
extensions, and their profile-scoped forms.

### Replacement

```go
type Replacement struct {
    Name               string // Package the directive targets
    Version            string // Targeted version; empty targets every version
    ReplacementName    string // Package used instead, when the replacement is a package
    ReplacementVersion string // Version of ReplacementName, when given
    Source             Source // Replacement that is not a package, such as a local directory
    Patch              string // Patch file applied to the package, as written
    Location           string // Opaque parser-defined identity within the manifest
}
```

`Replacements` records directives that substitute the package or source used
for another package, in source order. `Parse` does not apply them, so
`Dependencies` still reports each requirement as written. A `go.mod` replace
only takes effect when that file is the main module, which `Parse` cannot know.
`go.mod` replace directives are recorded with a `path` source for a local
directory and a `ReplacementName` otherwise. `Declarations` omit modules named
on the left side of a replace directive.

`package.json` records npm `overrides`, Yarn `resolutions` and the
`pnpm.overrides` and `pnpm.patchedDependencies` fields, block by block and in
source order within each block. `pnpm-workspace.yaml` is parsed as an npm
supplement for the `overrides` and `patchedDependencies` that pnpm 10 keeps
there. `Name` is the package a key targets: the last package of a nested npm
override, a Yarn path such as `parent/**/child` or a pnpm chain such as
`parent>child`. A version range in the key, as in `child@^1`, becomes
`Version`. A pnpm selector splits at its first `>` that is not a comparison,
so `parent>foo@>2` targets `foo` at `>2`. Values are kept as written: `npm:`
aliases become a `ReplacementName`, while a range such as Yarn's `npm:4.17.21`
stays the `ReplacementVersion` of the same package. Git, GitHub, URL and local
values become a `Source`, while patch files and Yarn `patch:` descriptors
become a `Patch`. A patch of the target itself leaves `ReplacementName` empty,
so it is only set when another package is used instead. Anything else,
including `$name` references and pnpm's `-`, is the `ReplacementVersion`.
Nested npm overrides are read 32 objects deep. Deeper objects are skipped.

`Cargo.toml` records `[patch.<source>]` entries, located by source and key,
and `[replace]` entries, whose package ID spec gives `Name` and `Version`. A
spec such as `https://github.com/example/foo#1.0.0` takes the name from the
URL. A `path`, `git` or `registry` key becomes the `Source` while a `version`
key or a version string becomes the `ReplacementVersion`. Values of any other
type are skipped.

### Metadata

```go
type Metadata struct {
    Summary                *string  // Short description, when kept separate
    Description            *string  // Description as written
    DescriptionContentType *string  // Declared media type of Description
    Keywords               []string // Keywords or tags
    Homepage               *string  // Declared home page URL
    Links                  []Link   // Other declared URLs
    Readme                 *string  // Declared README path, not read
}

type Link struct {
    Label string // The format's own term, such as "repository"
    URL   string // As written
}
```

`Metadata` holds descriptive package fields. A nil pointer or slice means the
field is absent, while a pointer to `""` or an empty `Keywords` slice means the
manifest declares it empty. `Links` is nil when no URL is declared. A declared
README stays a path: `Parse` does not read it, even when `FSRoot` is set.

`package.json` fills `Description`, `Keywords` and `Homepage`, with
`repository`, `bugs` and each `funding` entry as `Links`. A value of an
unexpected type is treated as absent rather than failing the parse. `.nuspec`
fills `Summary` and `Description` verbatim, `Keywords` from `tags` split on
white space, `Homepage` from `projectUrl` plus `Readme` from `readme`, with the
`repository` URL, `licenseUrl` and `iconUrl` as `Links`.

### Registry

```go
type Registry struct {
    URL      string // Registry URL, with any userinfo removed
    Scope    string // npm scope, Cargo source or Bundler source it applies to; empty for every package
    Replaces bool   // Used instead of the default rather than in addition to it
    File     string // Filename passed to Parse
}
```

These files often hold credentials. `Parse` never returns them: only registry
keys are read, so token, password and auth keys are skipped. Userinfo is also
removed from every URL and URL-shaped scope, so a result is safe to log or store.
Values are literal, so environment references such as `${NPM_REGISTRY}` are
kept as written. Cargo directory, local registry and git replacements are not
registries and are omitted; `DiscoverVendors` reports directory sources.

### ParseResult

```go
type ParseResult struct {
    Ecosystem    string       // npm, gem, pypi, golang, cargo, etc.
    Kind         Kind         // manifest, lockfile, supplement or config
    Name         string       // the package's own name, when the format declares one
    Version      string       // the package's own version, when declared
    Licenses     []string     // raw declared license values
    LicenseFile  string       // manifest-relative path to a declared license file
    Digest       string       // file-level verification value, when present
    Dependencies []Dependency
    Declarations []Declaration
    Replacements []Replacement // Replace or override directives, not applied
    Registries   []Registry    // Registry redirection configuration, without credentials
    Sources      []Source      // Ordered manifest-level source declarations
    Scripts      map[string][]string // Hook or task names with ordered commands or paths
    Publication  Publication   // Package publication settings
    Metadata     Metadata      // Descriptive package fields
}
```

`Name` and `Version` are populated for manifest formats that declare their own package identity (Cargo.toml `[package]`, package.json `"name"`, go.mod `module`, `.gemspec`, and so on). They are empty for lockfiles and for dependency-only files like Gemfile or requirements.txt.

`Licenses` contains decoded values as declared by the manifest; it does not normalize them into SPDX expressions. `LicenseFile` is populated when a format explicitly identifies a license file. Both are empty for formats without license metadata.

`Digest` contains a file-level verification value when the format defines one. For `Chart.lock`, it covers the dependency declarations from `Chart.yaml` and is separate from each dependency's `Integrity` value.

`Scripts` contains declared commands, callbacks or script paths, including named tasks as well as install hooks. For example, `result.Scripts["postinstall"]` holds the command from npm's `scripts.postinstall`. Lists preserve declaration order; empty declarations are omitted.

`Publication.Provenance` preserves an explicit package provenance setting. It is a pointer so callers can distinguish an omitted setting from `false`. The npm parsers read `publishConfig.provenance` and `.npmrc`'s `provenance` key; the value describes configuration and does not prove that a registry received an attestation.

| Format | Script declarations |
| --- | --- |
| `package.json`, `composer.json`, `shard.yml`, .NET `project.json` | Entries in `scripts` |
| `Cargo.toml` | `package.build`, under `build`; explicit `true` produces `build.rs`, `false` produces no entry |
| `.gemspec` | Literal `extensions` arrays, `%w` lists and `<<` additions |
| `.podspec` | Quoted or heredoc `prepare_command` |
| `deno.json` | Task strings and task objects' `command` values, under the task name |
| `dub.json` | Build, generate and run commands; platform suffixes are retained, with `configurations/<name>/`, `buildTypes/<name>/` and `subPackages/<name>/` prefixes for nested declarations |
| `.csproj`, `.vbproj`, `.fsproj` | `PreBuildEvent`, `PostBuildEvent` and direct `Exec` commands under `Target/<name>` |
| `pyproject.toml` | Poetry's build script under `tool.poetry.build`, and PDM's explicit `tool.pdm.build.custom-hook` |
| OPAM | `build`, `install`, `remove`, `run-test`, `build-test` and `build-doc`, each as a raw field value retaining arguments and filters |
| `APKBUILD`, `PKGBUILD` | Unindented `install` assignments, under `install`, without shell expansion |

Parsing does not execute scripts, evaluate conditions or follow script paths. Ruby string bodies retain source escapes and heredoc indentation. Dynamic Ruby expressions and arbitrary build code are not resolved. A missing entry does not establish that a package has no install hooks: files such as an undeclared `build.rs` or `deps/build.jl` require separate filesystem discovery.

Chef cookbook and Vagrant box PURLs remain empty while `chef` and `vagrant`
are only candidate Package URL types without accepted name and namespace rules.

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
    Config     Kind = "config"     // Package manager configuration, such as registry settings
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

## License

[MIT](LICENSE).
