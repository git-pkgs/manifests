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
| bower | bower.json | |
| brew | Brewfile | Brewfile.lock.json |
| cargo | Cargo.toml | Cargo.lock |
| carthage | Cartfile, Cartfile.private | Cartfile.resolved |
| clojars | project.clj | |
| cocoapods | Podfile, *.podspec | Podfile.lock |
| composer | composer.json | composer.lock |
| conan | conanfile.txt, conanfile.py | conan.lock |
| conda | environment.yml, environment.yaml | |
| cpan | cpanfile, Makefile.PL, Build.PL, dist.ini, META.json, META.yml | cpanfile.snapshot |
| cran | DESCRIPTION | renv.lock |
| crystal | shard.yml | shard.lock |
| deno | deno.json, deno.jsonc | deno.lock |
| docker | Dockerfile, docker-compose.yml | |
| dub | dub.json, dub.sdl | |
| elm | elm.json, elm-package.json | |
| gem | Gemfile, gems.rb, *.gemspec | Gemfile.lock, gems.locked |
| github-actions | .github/workflows/*.yml | |
| golang | go.mod, Godeps, glide.yaml, Gopkg.toml | Godeps.json, glide.lock, Gopkg.lock, vendor.json, go-resolved-dependencies.json, vendor/manifest |
| hackage | *.cabal | stack.yaml.lock, cabal.config, cabal.project.freeze |
| haxelib | haxelib.json | |
| hex | mix.exs, gleam.toml | mix.lock, rebar.lock |
| julia | Project.toml, REQUIRE | Manifest.toml |
| luarocks | *.rockspec | |
| maven | pom.xml, ivy.xml, build.gradle, build.gradle.kts, build.sbt | gradle.lockfile, gradle-dependencies-q.txt, maven-resolved-dependencies.txt, verification-metadata.xml |
| nimble | *.nimble | |
| nix | flake.nix | flake.lock, sources.json |
| npm | package.json, bower.json | package-lock.json, npm-shrinkwrap.json, yarn.lock, pnpm-lock.yaml, bun.lock, npm-ls.json |
| nuget | *.csproj, *.vbproj, *.fsproj, *.nuspec, packages.config, Project.json | packages.lock.json, paket.lock, project.assets.json, *.deps.json, Project.lock.json |
| pub | pubspec.yaml | pubspec.lock |
| pypi | requirements.txt, Pipfile, pyproject.toml, setup.py | Pipfile.lock, poetry.lock, pdm.lock, uv.lock, pip-dependency-graph.json, pip-resolved-dependencies.txt, pylock.toml |
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
| Gemfile.lock | ✓ | ✓ | | ✓ |
| Cargo.lock | ✓ | ✓ | | |
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
| rebar.lock | | | | |
| pubspec.lock | | | | |
| conan.lock | | | ✓ | |
| packages.lock.json | | | | ✓ |
| paket.lock | | | | |
| project.assets.json | | | | |
| *.deps.json | | ✓ | | |
| Project.lock.json | | ✓ | | |
| stack.yaml.lock | | | | |
| cabal.config | | | | |
| cabal.project.freeze | | | | |
| renv.lock | | ✓ | | |
| shard.lock | | | | |
| flake.lock | | | | |
| Brewfile.lock.json | | ✓ | | ✓ |

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

## Types

### Dependency

```go
type Dependency struct {
    Name        string // Package name
    Version     string // Version constraint or resolved version
    Scope       Scope  // runtime, development, test, build, optional
    Integrity   string // SRI hash (sha256-..., sha512-...)
    Direct      bool   // True if declared directly, false if transitive
    PURL        string // Package URL (pkg:ecosystem/name@version)
    RegistryURL string // Source registry URL (if non-default)
}
```

When a dependency comes from a non-default registry, the PURL includes a `repository_url` qualifier (e.g., `pkg:npm/foo@1.0.0?repository_url=https://npm.mycompany.com/`). Default registries like registry.npmjs.org, pypi.org, and rubygems.org are not included in the PURL.

### ParseResult

```go
type ParseResult struct {
    Ecosystem    string       // npm, gem, pypi, golang, cargo, etc.
    Kind         Kind         // manifest, lockfile, or supplement
    Dependencies []Dependency
}
```

### Kind

```go
const (
    Manifest   Kind = "manifest"   // Declared dependencies with version constraints
    Lockfile   Kind = "lockfile"   // Resolved dependencies with exact versions
    Supplement Kind = "supplement" // Provides extra data (e.g. integrity hashes) for a manifest's dependencies
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
