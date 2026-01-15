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
| golang | go.mod | go.sum |
| hackage | *.cabal | stack.yaml.lock, cabal.config, cabal.project.freeze |
| haxelib | haxelib.json | |
| hex | mix.exs, gleam.toml | mix.lock, rebar.lock |
| julia | Project.toml | Manifest.toml |
| luarocks | *.rockspec | |
| maven | pom.xml, ivy.xml, build.gradle, build.gradle.kts, build.sbt | gradle.lockfile |
| nimble | *.nimble | |
| nix | flake.nix | flake.lock |
| npm | package.json | package-lock.json, npm-shrinkwrap.json, yarn.lock, pnpm-lock.yaml, bun.lock |
| nuget | *.csproj, *.vbproj, *.fsproj, *.nuspec, packages.config | packages.lock.json, paket.lock |
| pub | pubspec.yaml | pubspec.lock |
| pypi | requirements.txt, Pipfile, pyproject.toml | Pipfile.lock, poetry.lock |
| rpm | *.spec | |
| swift | Package.swift | Package.resolved |
| vcpkg | vcpkg.json | |

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
    Name      string // Package name
    Version   string // Version constraint or resolved version
    Scope     Scope  // runtime, development, test, build, optional
    Integrity string // SRI hash (sha256-..., sha512-...)
    Direct    bool   // True if declared directly, false if transitive
    PURL      string // Package URL (pkg:ecosystem/name@version)
}
```

### ParseResult

```go
type ParseResult struct {
    Ecosystem    string       // npm, gem, pypi, golang, cargo, etc.
    Kind         Kind         // manifest or lockfile
    Dependencies []Dependency
}
```

### Kind

```go
const (
    Manifest Kind = "manifest" // Declared dependencies with version constraints
    Lockfile Kind = "lockfile" // Resolved dependencies with exact versions
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
