package core

import (
	"path/filepath"
	"strings"
)

// Registration holds parser metadata.
type Registration struct {
	Ecosystem string
	Kind      Kind
	Parser    Parser
	Match     func(filename string) bool
}

var parsers []Registration

// Register adds a parser to the registry.
func Register(ecosystem string, kind Kind, parser Parser, match func(string) bool) {
	parsers = append(parsers, Registration{
		Ecosystem: ecosystem,
		Kind:      kind,
		Parser:    parser,
		Match:     match,
	})
}

// IdentifyParser returns the first matching parser for a filename.
func IdentifyParser(filename string) (Parser, string, Kind) {
	base := filepath.Base(filename)
	for _, reg := range parsers {
		if reg.Match(filename) || reg.Match(base) {
			return reg.Parser, reg.Ecosystem, reg.Kind
		}
	}
	return nil, "", ""
}

// Match represents a file type match.
type Match struct {
	Ecosystem string
	Kind      Kind
}

// IdentifyAllParsers returns all matching parsers for a filename.
func IdentifyAllParsers(filename string) []Match {
	base := filepath.Base(filename)
	var matches []Match
	for _, reg := range parsers {
		if reg.Match(filename) || reg.Match(base) {
			matches = append(matches, Match{
				Ecosystem: reg.Ecosystem,
				Kind:      reg.Kind,
			})
		}
	}
	return matches
}

// SupportedEcosystems returns all registered ecosystem types.
func SupportedEcosystems() []string {
	seen := make(map[string]bool)
	var ecosystems []string
	for _, reg := range parsers {
		if !seen[reg.Ecosystem] {
			seen[reg.Ecosystem] = true
			ecosystems = append(ecosystems, reg.Ecosystem)
		}
	}
	return ecosystems
}

// ExactMatch returns a matcher for exact filename matches.
func ExactMatch(names ...string) func(string) bool {
	set := make(map[string]bool)
	for _, name := range names {
		set[name] = true
	}
	return func(filename string) bool {
		return set[filename]
	}
}

// SuffixMatch returns a matcher for suffix matches.
func SuffixMatch(suffixes ...string) func(string) bool {
	return func(filename string) bool {
		for _, suffix := range suffixes {
			if strings.HasSuffix(filename, suffix) {
				return true
			}
		}
		return false
	}
}

// PrefixMatch returns a matcher for prefix matches.
func PrefixMatch(prefixes ...string) func(string) bool {
	return func(filename string) bool {
		for _, prefix := range prefixes {
			if strings.HasPrefix(filename, prefix) {
				return true
			}
		}
		return false
	}
}

// GlobMatch returns a matcher for glob pattern matches.
func GlobMatch(pattern string) func(string) bool {
	return func(filename string) bool {
		matched, _ := filepath.Match(pattern, filename)
		return matched
	}
}

// AnyMatch returns a matcher that matches if any of the given matchers match.
func AnyMatch(matchers ...func(string) bool) func(string) bool {
	return func(filename string) bool {
		for _, m := range matchers {
			if m(filename) {
				return true
			}
		}
		return false
	}
}
