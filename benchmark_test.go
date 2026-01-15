package manifests

import (
	"os"
	"path/filepath"
	"testing"
)

// Benchmark fixtures grouped by parser type
var benchmarkFixtures = map[string][]string{
	// JSON parsers
	"json": {
		"testdata/npm/package.json",
		"testdata/npm/package-lock.json",
		"testdata/composer/composer.json",
		"testdata/composer/composer.lock",
		"testdata/npm/deno.json",
		"testdata/npm/deno.lock",
		"testdata/npm/bun.lock",
		"testdata/vcpkg/vcpkg.json",
	},
	// YAML parsers
	"yaml": {
		"testdata/pub/pubspec.yaml",
		"testdata/pub/pubspec.lock",
		"testdata/pypi/environment.yml",
		"testdata/npm/pnpm-lock.yaml",
		"testdata/docker/docker-compose.yml",
		"testdata/crystal/shard.yml",
		"testdata/crystal/shard.lock",
	},
	// TOML parsers
	"toml": {
		"testdata/cargo/Cargo.toml",
		"testdata/cargo/Cargo.lock",
		"testdata/pypi/pyproject.toml",
		"testdata/gleam/gleam.toml",
		"testdata/julia/Project.toml",
		"testdata/julia/Manifest.toml",
	},
	// XML parsers
	"xml": {
		"testdata/maven/pom.xml",
		"testdata/maven/ivy.xml",
		"testdata/nuget/example.csproj",
		"testdata/nuget/packages.config",
	},
	// Line-based/regex parsers
	"regex": {
		"testdata/gem/Gemfile",
		"testdata/gem/Gemfile.lock",
		"testdata/pypi/requirements.txt",
		"testdata/golang/go.mod",
		"testdata/golang/go.sum",
		"testdata/npm/yarn.lock",
		"testdata/docker/Dockerfile",
		"testdata/maven/build.gradle",
		"testdata/hackage/example.cabal",
		"testdata/cpan/cpanfile",
	},
}

func BenchmarkParse(b *testing.B) {
	// Collect all fixtures
	var fixtures []string
	for _, files := range benchmarkFixtures {
		fixtures = append(fixtures, files...)
	}

	for _, fixture := range fixtures {
		content, err := os.ReadFile(fixture)
		if err != nil {
			continue // Skip missing files
		}

		name := filepath.Base(fixture)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))

			for i := 0; i < b.N; i++ {
				_, err := Parse(name, content)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseByType(b *testing.B) {
	for parserType, fixtures := range benchmarkFixtures {
		b.Run(parserType, func(b *testing.B) {
			// Load all fixtures for this type
			type fixture struct {
				name    string
				content []byte
			}
			var loaded []fixture

			for _, path := range fixtures {
				content, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				loaded = append(loaded, fixture{
					name:    filepath.Base(path),
					content: content,
				})
			}

			if len(loaded) == 0 {
				b.Skip("no fixtures found")
			}

			b.ReportAllocs()

			var totalBytes int64
			for _, f := range loaded {
				totalBytes += int64(len(f.content))
			}
			b.SetBytes(totalBytes)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				for _, f := range loaded {
					_, err := Parse(f.name, f.content)
					if err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkIdentify(b *testing.B) {
	filenames := []string{
		"package.json",
		"package-lock.json",
		"Gemfile",
		"Gemfile.lock",
		"requirements.txt",
		"go.mod",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"pubspec.yaml",
		".github/workflows/ci.yml",
		"unknown.txt",
	}

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, name := range filenames {
			Identify(name)
		}
	}
}

func BenchmarkIdentifyAll(b *testing.B) {
	filenames := []string{
		"package.json",
		"Gemfile",
		"requirements.txt",
		"pom.xml",
	}

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, name := range filenames {
			IdentifyAll(name)
		}
	}
}

// BenchmarkLargeFiles tests parsing performance on larger files
func BenchmarkLargeFiles(b *testing.B) {
	largeFiles := []string{
		"testdata/gem/Gemfile.lock",
		"testdata/npm/package-lock.json",
		"testdata/npm/yarn.lock",
		"testdata/npm/pnpm-lock.yaml",
		"testdata/cargo/Cargo.lock",
		"testdata/golang/go.sum",
		"testdata/composer/composer.lock",
		"testdata/nuget/packages.lock.json",
	}

	for _, path := range largeFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Only benchmark files > 1KB
		if len(content) < 1024 {
			continue
		}

		name := filepath.Base(path)
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(content)))

			for i := 0; i < b.N; i++ {
				_, err := Parse(name, content)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
