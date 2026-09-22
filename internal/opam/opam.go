package opam

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/git-pkgs/manifests/internal/core"
)

var opamVersionConstraintPattern = regexp.MustCompile(`(>=|<=|!=|=|>|<)\s*("(?:\\.|[^"\\])*")`)

func init() {
	core.Register("opam", core.Manifest, &parser{},
		core.AnyMatch(core.ExactMatch("opam"), core.SuffixMatch(".opam")))
}

// parser parses OPAM package definition files.
type parser struct{}

func (p *parser) Parse(_ string, content []byte) (*core.Result, error) {
	text := string(content)

	return &core.Result{
		Name:         opamScalar(opamField(text, "name")),
		Version:      opamScalar(opamField(text, "version")),
		Licenses:     opamTopLevelStrings(opamField(text, "license")),
		Scripts:      opamScripts(text),
		Dependencies: opamDependencies(opamField(text, "depends")),
	}, nil
}

func opamScripts(text string) map[string][]string {
	var scripts map[string][]string
	for _, field := range []string{"build", "install", "remove", "run-test", "build-test", "build-doc"} {
		value := opamField(text, field)
		if value != "[]" && strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")) != "" {
			// Keep argument boundaries, filters, and variable references intact.
			core.AddScript(&scripts, field, value)
		}
	}
	return scripts
}

func opamField(text, field string) string {
	pattern := regexp.MustCompile(`(?m)^[ \t]*` + regexp.QuoteMeta(field) + `[ \t]*:[ \t]*`)
	location := pattern.FindStringIndex(text)
	if location == nil {
		return ""
	}

	start := skipHorizontalSpace(text, location[1])
	if start >= len(text) {
		return ""
	}
	if text[start] != '[' {
		return opamScalarField(text[start:])
	}

	_, end := opamDelimited(text, start, '[', ']')
	return strings.TrimSpace(text[start:end])
}

func skipHorizontalSpace(value string, start int) int {
	for start < len(value) && (value[start] == ' ' || value[start] == '\t') {
		start++
	}
	return start
}

func opamScalarField(value string) string {
	if end := strings.IndexByte(value, '\n'); end >= 0 {
		value = value[:end]
	}
	return strings.TrimSpace(value)
}

type opamLexState struct {
	inString  bool
	escaped   bool
	inComment bool
}

func (s *opamLexState) consume(char byte) bool {
	switch {
	case s.inComment:
		if char == '\n' {
			s.inComment = false
		}
		return true
	case s.inString:
		s.consumeStringChar(char)
		return true
	}

	switch char {
	case '#':
		s.inComment = true
		return true
	case '"':
		s.inString = true
		return true
	default:
		return false
	}
}

func (s *opamLexState) consumeStringChar(char byte) {
	switch {
	case s.escaped:
		s.escaped = false
	case char == '\\':
		s.escaped = true
	case char == '"':
		s.inString = false
	}
}

func opamDelimited(value string, start int, open, close byte) (string, int) {
	depth := 0
	var state opamLexState
	for i := start; i < len(value); i++ {
		if state.consume(value[i]) {
			continue
		}
		switch value[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return value[start+1 : i], i + 1
			}
		}
	}
	return value[start+1:], len(value)
}

func opamScalar(value string) string {
	values := opamTopLevelStrings(value)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

type opamEntry struct {
	value   string
	formula string
}

type opamEntryScanner struct {
	value string
	pos   int
}

func (s *opamEntryScanner) next() (opamEntry, bool) {
	for s.pos < len(s.value) {
		s.skipTrivia()
		if s.pos >= len(s.value) {
			break
		}

		switch s.value[s.pos] {
		case '"':
			return s.quotedEntry(), true
		case '{':
			_, s.pos = opamDelimited(s.value, s.pos, '{', '}')
		default:
			s.pos++
		}
	}
	return opamEntry{}, false
}

func (s *opamEntryScanner) skipTrivia() {
	for s.pos < len(s.value) {
		switch s.value[s.pos] {
		case ' ', '\t', '\r', '\n':
			s.pos++
		case '#':
			s.skipComment()
		default:
			return
		}
	}
}

func (s *opamEntryScanner) skipComment() {
	end := strings.IndexByte(s.value[s.pos:], '\n')
	if end < 0 {
		s.pos = len(s.value)
		return
	}
	s.pos += end + 1
}

func (s *opamEntryScanner) quotedEntry() opamEntry {
	value, next := opamQuotedString(s.value, s.pos)
	s.pos = next
	s.skipTrivia()

	entry := opamEntry{value: value}
	var formulas []string
	for s.pos < len(s.value) && s.value[s.pos] == '{' {
		formula, next := opamDelimited(s.value, s.pos, '{', '}')
		formulas = append(formulas, formula)
		s.pos = next
		s.skipTrivia()
	}
	entry.formula = strings.Join(formulas, " & ")
	return entry
}

func opamQuotedString(value string, start int) (string, int) {
	end := start + 1
	for end < len(value) {
		switch value[end] {
		case '\\':
			end += 2
			continue
		case '"':
			return unquoteOpamString(value[start : end+1]), end + 1
		}
		end++
	}
	return strings.TrimPrefix(value[start:], `"`), len(value)
}

func unquoteOpamString(value string) string {
	unquoted, err := strconv.Unquote(value)
	if err != nil {
		return strings.Trim(value, `"`)
	}
	return unquoted
}

func opamEntries(value string) []opamEntry {
	scanner := opamEntryScanner{value: value}
	var entries []opamEntry
	for {
		entry, ok := scanner.next()
		if !ok {
			return entries
		}
		entries = append(entries, entry)
	}
}

func opamTopLevelStrings(value string) []string {
	entries := opamEntries(value)
	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		values = append(values, entry.value)
	}
	return values
}

func opamDependencies(value string) []core.Dependency {
	entries := opamEntries(value)
	dependencies := make([]core.Dependency, 0, len(entries))
	for _, entry := range entries {
		dependencies = append(dependencies, core.Dependency{
			Name:    entry.value,
			Version: opamDependencyVersion(entry.formula),
			Scope:   opamDependencyScope(entry.formula),
			Direct:  true,
		})
	}
	return dependencies
}

func opamDependencyVersion(formula string) string {
	matches := opamVersionConstraintPattern.FindAllStringSubmatchIndex(formula, -1)
	var constraints []string
	previousEnd := 0
	for _, match := range matches {
		if !standaloneOpamConstraint(formula, match[0]) {
			continue
		}
		if len(constraints) > 0 {
			constraints = append(constraints, opamConstraintConnector(formula[previousEnd:match[0]]))
		}
		operator := formula[match[2]:match[3]]
		version := unquoteOpamString(formula[match[4]:match[5]])
		constraints = append(constraints, operator+" "+version)
		previousEnd = match[1]
	}
	return strings.Join(constraints, " ")
}

func standaloneOpamConstraint(formula string, start int) bool {
	prefix := strings.TrimSpace(formula[:start])
	if prefix == "" {
		return true
	}
	switch prefix[len(prefix)-1] {
	case '(', '&', '|', '!':
		return true
	default:
		return false
	}
}

func opamConstraintConnector(value string) string {
	if strings.Contains(value, "|") {
		return "|"
	}
	return "&"
}

func opamDependencyScope(formula string) core.Scope {
	formula = opamVersionConstraintPattern.ReplaceAllString(formula, "")
	var build, test, development bool
	for _, field := range strings.FieldsFunc(formula, func(char rune) bool {
		return !isOpamVariableChar(char)
	}) {
		switch field {
		case "with-test":
			test = true
		case "build":
			build = true
		case "with-doc", "with-dev-setup":
			development = true
		}
	}

	switch {
	case test:
		return core.Test
	case build:
		return core.Build
	case development:
		return core.Development
	default:
		return core.Runtime
	}
}

func isOpamVariableChar(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_'
}
