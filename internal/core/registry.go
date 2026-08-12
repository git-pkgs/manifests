package core

import (
	"math/bits"
	"path/filepath"
	"strings"
)

const (
	candidateWordBits   = 64
	stackCandidateWords = 4
)

// Registration holds parser metadata.
type Registration struct {
	Ecosystem string
	Kind      Kind
	Parser    Parser
	Match     func(filename string) bool
}

// Matcher describes filename matching rules that the registry can index.
type Matcher struct {
	exact  []string
	suffix []string
	prefix []string
	glob   []globPattern
	custom []func(string) bool
}

type globPattern struct {
	pattern string
	prefix  string
	suffix  string
	literal string
}

type patternRegistration struct {
	registration int
	patterns     []string
}

type globRegistration struct {
	registration int
	patterns     []globPattern
}

type customRegistration struct {
	registration int
	matchers     []func(string) bool
}

type parserRegistry struct {
	registrations []Registration
	exact         map[string][]int
	suffix        []patternRegistration
	prefix        []patternRegistration
	glob          []globRegistration
	custom        []customRegistration
}

var defaultRegistry parserRegistry

// Register adds a parser to the registry.
func Register(ecosystem string, kind Kind, parser Parser, matcher Matcher) {
	defaultRegistry.register(ecosystem, kind, parser, matcher)
}

func (r *parserRegistry) register(ecosystem string, kind Kind, parser Parser, matcher Matcher) {
	registration := len(r.registrations)
	r.registrations = append(r.registrations, Registration{
		Ecosystem: ecosystem,
		Kind:      kind,
		Parser:    parser,
		Match:     matcher.Match,
	})

	if len(matcher.exact) > 0 {
		if r.exact == nil {
			r.exact = make(map[string][]int)
		}
		for _, name := range matcher.exact {
			registrations := r.exact[name]
			if len(registrations) == 0 || registrations[len(registrations)-1] != registration {
				r.exact[name] = append(registrations, registration)
			}
		}
	}
	if len(matcher.suffix) > 0 {
		r.suffix = append(r.suffix, patternRegistration{registration: registration, patterns: matcher.suffix})
	}
	if len(matcher.prefix) > 0 {
		r.prefix = append(r.prefix, patternRegistration{registration: registration, patterns: matcher.prefix})
	}
	if len(matcher.glob) > 0 {
		r.glob = append(r.glob, globRegistration{registration: registration, patterns: matcher.glob})
	}
	if len(matcher.custom) > 0 {
		r.custom = append(r.custom, customRegistration{registration: registration, matchers: matcher.custom})
	}
}

// IdentifyParser returns the first matching parser for a filename.
func IdentifyParser(filename string) (Parser, string, Kind) { //nolint:ireturn
	return defaultRegistry.identifyParser(filename)
}

func (r *parserRegistry) identifyParser(filename string) (Parser, string, Kind) { //nolint:ireturn
	base := filepath.Base(filename)
	best := len(r.registrations)

	if registrations := r.exact[filename]; len(registrations) > 0 {
		best = registrations[0]
	}
	if base != filename {
		if registrations := r.exact[base]; len(registrations) > 0 && registrations[0] < best {
			best = registrations[0]
		}
	}

	for _, candidate := range r.suffix {
		if candidate.registration >= best {
			break
		}
		if matchesSuffix(filename, base, candidate.patterns) {
			best = candidate.registration
		}
	}
	for _, candidate := range r.prefix {
		if candidate.registration >= best {
			break
		}
		if matchesPrefix(filename, base, candidate.patterns) {
			best = candidate.registration
		}
	}
	for _, candidate := range r.glob {
		if candidate.registration >= best {
			break
		}
		if matchesGlob(filename, base, candidate.patterns) {
			best = candidate.registration
		}
	}
	for _, candidate := range r.custom {
		if candidate.registration >= best {
			break
		}
		if matchesCustom(filename, base, candidate.matchers) {
			best = candidate.registration
		}
	}

	if best == len(r.registrations) {
		return nil, "", ""
	}
	registration := r.registrations[best]
	return registration.Parser, registration.Ecosystem, registration.Kind
}

// Match represents a file type match.
type Match struct {
	Ecosystem string
	Kind      Kind
}

// IdentifyAllParsers returns all matching parsers for a filename.
func IdentifyAllParsers(filename string) []Match {
	return defaultRegistry.identifyAllParsers(filename)
}

func (r *parserRegistry) identifyAllParsers(filename string) []Match {
	base := filepath.Base(filename)
	wordCount := (len(r.registrations) + candidateWordBits - 1) / candidateWordBits
	var stackCandidates [stackCandidateWords]uint64
	var candidates []uint64
	if wordCount > len(stackCandidates) {
		candidates = make([]uint64, wordCount)
	} else {
		candidates = stackCandidates[:wordCount]
	}

	markRegistrations(candidates, r.exact[filename])
	if base != filename {
		markRegistrations(candidates, r.exact[base])
	}
	for _, candidate := range r.suffix {
		if matchesSuffix(filename, base, candidate.patterns) {
			markRegistration(candidates, candidate.registration)
		}
	}
	for _, candidate := range r.prefix {
		if matchesPrefix(filename, base, candidate.patterns) {
			markRegistration(candidates, candidate.registration)
		}
	}
	for _, candidate := range r.glob {
		if matchesGlob(filename, base, candidate.patterns) {
			markRegistration(candidates, candidate.registration)
		}
	}
	for _, candidate := range r.custom {
		if matchesCustom(filename, base, candidate.matchers) {
			markRegistration(candidates, candidate.registration)
		}
	}

	var matches []Match
	for wordIndex, word := range candidates {
		for word != 0 {
			bit := bits.TrailingZeros64(word)
			registration := r.registrations[wordIndex*candidateWordBits+bit]
			matches = append(matches, Match{
				Ecosystem: registration.Ecosystem,
				Kind:      registration.Kind,
			})
			word &= word - 1
		}
	}
	return matches
}

func markRegistrations(candidates []uint64, registrations []int) {
	for _, registration := range registrations {
		markRegistration(candidates, registration)
	}
}

func markRegistration(candidates []uint64, registration int) {
	candidates[registration/candidateWordBits] |= 1 << (registration % candidateWordBits)
}

// SupportedEcosystems returns all registered ecosystem types.
func SupportedEcosystems() []string {
	seen := make(map[string]bool)
	var ecosystems []string
	for _, registration := range defaultRegistry.registrations {
		if !seen[registration.Ecosystem] {
			seen[registration.Ecosystem] = true
			ecosystems = append(ecosystems, registration.Ecosystem)
		}
	}
	return ecosystems
}

// ExactMatch returns a matcher for exact filename matches.
func ExactMatch(names ...string) Matcher {
	return Matcher{exact: append([]string(nil), names...)}
}

// SuffixMatch returns a matcher for suffix matches.
func SuffixMatch(suffixes ...string) Matcher {
	return Matcher{suffix: suffixes}
}

// PrefixMatch returns a matcher for prefix matches.
func PrefixMatch(prefixes ...string) Matcher {
	return Matcher{prefix: prefixes}
}

// GlobMatch returns a matcher for glob pattern matches.
func GlobMatch(pattern string) Matcher {
	prefix, suffix, literal := globParts(pattern)
	return Matcher{glob: []globPattern{{pattern: pattern, prefix: prefix, suffix: suffix, literal: literal}}}
}

// CustomMatch returns a matcher backed by a custom matching function.
func CustomMatch(match func(string) bool) Matcher {
	return Matcher{custom: []func(string) bool{match}}
}

// AnyMatch returns a matcher that matches if any of the given matchers match.
func AnyMatch(matchers ...Matcher) Matcher {
	var combined Matcher
	for _, matcher := range matchers {
		combined.exact = append(combined.exact, matcher.exact...)
		combined.suffix = append(combined.suffix, matcher.suffix...)
		combined.prefix = append(combined.prefix, matcher.prefix...)
		combined.glob = append(combined.glob, matcher.glob...)
		combined.custom = append(combined.custom, matcher.custom...)
	}
	return combined
}

// Match reports whether the matcher matches a filename.
func (m Matcher) Match(filename string) bool {
	for _, name := range m.exact {
		if filename == name {
			return true
		}
	}
	if matchesSuffix(filename, filename, m.suffix) {
		return true
	}
	if matchesPrefix(filename, filename, m.prefix) {
		return true
	}
	if matchesGlob(filename, filename, m.glob) {
		return true
	}
	return matchesCustom(filename, filename, m.custom)
}

func matchesSuffix(filename, base string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(filename, suffix) || base != filename && strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

func matchesPrefix(filename, base string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(filename, prefix) || base != filename && strings.HasPrefix(base, prefix) {
			return true
		}
	}
	return false
}

func matchesGlob(filename, base string, patterns []globPattern) bool {
	for _, pattern := range patterns {
		if globMatches(pattern, filename) || base != filename && globMatches(pattern, base) {
			return true
		}
	}
	return false
}

func globMatches(pattern globPattern, filename string) bool {
	if pattern.prefix != "" && !strings.HasPrefix(filename, pattern.prefix) {
		return false
	}
	if pattern.suffix != "" && !strings.HasSuffix(filename, pattern.suffix) {
		return false
	}
	if pattern.literal != "" && !strings.Contains(filename, pattern.literal) {
		return false
	}
	matched, _ := filepath.Match(pattern.pattern, filename)
	return matched
}

func globParts(pattern string) (prefix, suffix, literal string) {
	if strings.ContainsAny(pattern, "[\\") {
		return "", "", ""
	}
	firstMeta := strings.IndexAny(pattern, "*?")
	if firstMeta < 0 {
		return pattern, pattern, ""
	}
	lastMeta := strings.LastIndexAny(pattern, "*?")
	prefix = pattern[:firstMeta]
	suffix = pattern[lastMeta+1:]

	for remainder := pattern[firstMeta : lastMeta+1]; remainder != ""; {
		start := strings.IndexAny(remainder, "*?")
		if start < 0 {
			break
		}
		remainder = remainder[start+1:]
		end := strings.IndexAny(remainder, "*?")
		if end < 0 {
			end = len(remainder)
		}
		if end > len(literal) {
			literal = remainder[:end]
		}
		if end == len(remainder) {
			break
		}
		remainder = remainder[end:]
	}
	return prefix, suffix, literal
}

func matchesCustom(filename, base string, matchers []func(string) bool) bool {
	for _, matcher := range matchers {
		if matcher(filename) || base != filename && matcher(base) {
			return true
		}
	}
	return false
}
