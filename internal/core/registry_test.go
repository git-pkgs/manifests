package core

import (
	"reflect"
	"testing"
)

type registryTestParser struct{}

func (p *registryTestParser) Parse(string, []byte) (*Result, error) {
	return &Result{}, nil
}

func TestMatcherHelpers(t *testing.T) {
	tests := []struct {
		name     string
		matcher  Matcher
		filename string
		want     bool
	}{
		{name: "exact", matcher: ExactMatch("package.json"), filename: "package.json", want: true},
		{name: "exact mismatch", matcher: ExactMatch("package.json"), filename: "other.json"},
		{name: "suffix", matcher: SuffixMatch(".gemspec"), filename: "example.gemspec", want: true},
		{name: "prefix", matcher: PrefixMatch("Dockerfile."), filename: "Dockerfile.dev", want: true},
		{name: "glob", matcher: GlobMatch("requirements/*.txt"), filename: "requirements/dev.txt", want: true},
		{name: "glob character class", matcher: GlobMatch("file.[ch]"), filename: "file.c", want: true},
		{
			name: "any custom",
			matcher: AnyMatch(
				ExactMatch("manifest.scm"),
				CustomMatch(func(filename string) bool { return filename == "custom.lock" }),
			),
			filename: "custom.lock",
			want:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.matcher.Match(test.filename); got != test.want {
				t.Fatalf("Match(%q) = %v, want %v", test.filename, got, test.want)
			}
		})
	}
}

func TestParserRegistryPreservesPrecedenceAndOverlaps(t *testing.T) {
	var registry parserRegistry
	parser := &registryTestParser{}
	registry.register("suffix", Manifest, parser, SuffixMatch(".json"))
	registry.register("exact", Lockfile, parser, ExactMatch("package.json"))
	registry.register("custom", Supplement, parser, CustomMatch(func(filename string) bool {
		return filename == "services/package.json"
	}))

	_, ecosystem, kind := registry.identifyParser("services/package.json")
	if ecosystem != "suffix" || kind != Manifest {
		t.Fatalf("identifyParser() = %q, %q, want suffix, manifest", ecosystem, kind)
	}

	want := []Match{
		{Ecosystem: "suffix", Kind: Manifest},
		{Ecosystem: "exact", Kind: Lockfile},
		{Ecosystem: "custom", Kind: Supplement},
	}
	if got := registry.identifyAllParsers("services/package.json"); !reflect.DeepEqual(got, want) {
		t.Fatalf("identifyAllParsers() = %#v, want %#v", got, want)
	}
}

func TestParserRegistryReturnsCombinedMatcherOnce(t *testing.T) {
	var registry parserRegistry
	registry.register("pypi", Manifest, &registryTestParser{}, AnyMatch(
		ExactMatch("requirements.txt"),
		GlobMatch("*requirements*.txt"),
	))

	want := []Match{{Ecosystem: "pypi", Kind: Manifest}}
	if got := registry.identifyAllParsers("requirements.txt"); !reflect.DeepEqual(got, want) {
		t.Fatalf("identifyAllParsers() = %#v, want %#v", got, want)
	}
}

func TestParserRegistryMatchesFullPathAndBasename(t *testing.T) {
	tests := []struct {
		name      string
		matcher   Matcher
		filename  string
		ecosystem string
	}{
		{name: "exact basename", matcher: ExactMatch("go.mod"), filename: "services/api/go.mod", ecosystem: "exact"},
		{name: "prefix basename", matcher: PrefixMatch("Dockerfile."), filename: "containers/Dockerfile.dev", ecosystem: "prefix"},
		{name: "glob full path", matcher: GlobMatch("requirements/*.txt"), filename: "requirements/dev.txt", ecosystem: "glob"},
		{name: "suffix full path", matcher: SuffixMatch("vendor/manifest"), filename: "project/vendor/manifest", ecosystem: "suffix"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var registry parserRegistry
			registry.register(test.ecosystem, Manifest, &registryTestParser{}, test.matcher)

			_, ecosystem, _ := registry.identifyParser(test.filename)
			if ecosystem != test.ecosystem {
				t.Fatalf("identifyParser(%q) ecosystem = %q, want %q", test.filename, ecosystem, test.ecosystem)
			}
		})
	}
}
