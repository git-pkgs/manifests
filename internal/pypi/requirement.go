package pypi

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

// pep508Requirement is a PEP 508 requirement split into its parts. Each part
// is kept as written, without normalization or marker evaluation.
type pep508Requirement struct {
	Name    string
	Extras  []string
	Version string
	Marker  string
	Text    string
}

// parsePEP508Requirement splits a PEP 508 requirement such as
// "requests[socks] (>=2); python_version < '3.12'". A parenthesized specifier
// is unwrapped. A direct reference such as "pkg @ https://..." has no
// version. It reports false when text does not start with a name followed by
// extras, a specifier, a direct reference or a marker.
func parsePEP508Requirement(text string) (pep508Requirement, bool) {
	text = strings.TrimSpace(text)
	requirement := pep508Requirement{Text: text}

	end := 0
	for end < len(text) && isPEP508NameByte(text[end]) {
		end++
	}
	if end == 0 {
		return requirement, false
	}
	requirement.Name = text[:end]
	rest := trimPEP508Space(text[end:])

	if strings.HasPrefix(rest, "[") {
		closing := strings.IndexByte(rest, ']')
		if closing < 0 {
			return requirement, false
		}
		for extra := range strings.SplitSeq(rest[1:closing], ",") {
			if extra = strings.TrimSpace(extra); extra != "" {
				requirement.Extras = append(requirement.Extras, extra)
			}
		}
		rest = trimPEP508Space(rest[closing+1:])
	}

	if strings.HasPrefix(rest, "@") {
		// The URL ends at white space, so a ";" inside it is not a marker.
		rest = trimPEP508Space(rest[1:])
		urlEnd := strings.IndexAny(rest, " \t")
		if urlEnd < 0 {
			urlEnd = len(rest)
		}
		rest = trimPEP508Space(rest[urlEnd:])
		if rest != "" && rest[0] != ';' {
			return requirement, false
		}
		if marker, ok := strings.CutPrefix(rest, ";"); ok {
			requirement.Marker = strings.TrimSpace(marker)
		}
		return requirement, true
	}

	specifier, marker, hasMarker := strings.Cut(rest, ";")
	specifier = strings.TrimSpace(specifier)
	if strings.HasPrefix(specifier, "(") {
		specifier = strings.TrimSpace(strings.TrimSuffix(specifier[1:], ")"))
	}
	if specifier != "" && !strings.ContainsRune("<>=!~", rune(specifier[0])) {
		return requirement, false
	}
	requirement.Version = specifier
	if hasMarker {
		requirement.Marker = strings.TrimSpace(marker)
	}
	return requirement, true
}

func isPEP508NameByte(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '.' || c == '-' || c == '_'
}

func trimPEP508Space(value string) string {
	return strings.TrimLeft(value, " \t")
}

// dependency returns the requirement as a direct dependency in scope.
func (r pep508Requirement) dependency(scope core.Scope) core.Dependency {
	return core.Dependency{
		Name:    r.Name,
		Version: r.Version,
		Scope:   scope,
		Direct:  true,
		Extras:  r.Extras,
		Marker:  r.Marker,
	}
}

// appendPEP508 appends the requirement as a dependency and, when declarations
// is not nil, as a declaration that keeps the original text.
func appendPEP508(
	deps *[]core.Dependency,
	declarations *[]core.Declaration,
	locations map[string]int,
	prefix string,
	requirement pep508Requirement,
	scope core.Scope,
) {
	*deps = append(*deps, requirement.dependency(scope))
	if declarations == nil {
		return
	}
	appendPyPIDeclaration(declarations, locations, prefix, requirement.Name, requirement.Version, scope)
	declaration := &(*declarations)[len(*declarations)-1]
	declaration.Extras = requirement.Extras
	declaration.Marker = requirement.Marker
	declaration.Requirement = requirement.Text
}

// pythonStringList returns the string literals of the Python list that opens
// at content[start], which must be "[", plus the index of its closing "]". It
// honours quotes, so a "]" or the other quote character inside a string does
// not end it. Comments and anything that is not a string literal are skipped.
func pythonStringList(content string, start int) ([]string, int) {
	var values []string
	depth := 0
	for index := start; index < len(content); index++ {
		switch c := content[index]; c {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return values, index
			}
		case '#':
			newline := strings.IndexByte(content[index:], '\n')
			if newline < 0 {
				return values, len(content)
			}
			index += newline
		case '"', '\'':
			value, next, ok := pythonStringLiteral(content, index)
			if !ok {
				return values, len(content)
			}
			if depth == 1 {
				values = append(values, value)
			}
			index = next - 1
		}
	}
	return values, len(content)
}

// pythonStringLiteral reads the single, double or triple quoted string that
// starts at content[start] and returns its body and the index after it.
func pythonStringLiteral(content string, start int) (string, int, bool) {
	quote := content[start : start+1]
	if strings.HasPrefix(content[start:], strings.Repeat(quote, 3)) {
		quote = strings.Repeat(quote, 3)
	}
	body := start + len(quote)
	for index := body; index < len(content); index++ {
		if content[index] == '\\' {
			index++
			continue
		}
		if strings.HasPrefix(content[index:], quote) {
			return content[body:index], index + len(quote), true
		}
		if len(quote) == 1 && content[index] == '\n' {
			return "", 0, false
		}
	}
	return "", 0, false
}

// setupKeywordValue returns the index of the value of the first
// "keyword = <opener>" in content, such as the "[" of install_requires, or -1
// when there is none. Assignments to other values are passed over.
func setupKeywordValue(content, keyword string, opener byte) int {
	for offset := 0; ; {
		index := strings.Index(content[offset:], keyword)
		if index < 0 {
			return -1
		}
		index += offset
		offset = index + len(keyword)
		if index > 0 && isPEP508NameByte(content[index-1]) {
			continue
		}
		rest := strings.TrimLeft(content[offset:], " \t\r\n")
		if !strings.HasPrefix(rest, "=") || strings.HasPrefix(rest, "==") {
			continue
		}
		value := strings.TrimLeft(rest[1:], " \t\r\n")
		if value != "" && value[0] == opener {
			return len(content) - len(value)
		}
	}
}

// setupExtrasRequire returns the groups of a setup.py extras_require dict
// literal opening at content[start], in source order.
func setupExtrasRequire(content string, start int) []extrasGroup {
	var groups []extrasGroup
	for index := start + 1; index < len(content); index++ {
		switch c := content[index]; c {
		case '}':
			return groups
		case '#':
			if newline := strings.IndexByte(content[index:], '\n'); newline >= 0 {
				index += newline
			} else {
				return groups
			}
		case '"', '\'':
			name, next, ok := pythonStringLiteral(content, index)
			if !ok {
				return groups
			}
			rest := strings.TrimLeft(content[next:], " \t\r\n")
			if !strings.HasPrefix(rest, ":") {
				index = next - 1
				continue
			}
			rest = strings.TrimLeft(rest[1:], " \t\r\n")
			listStart := len(content) - len(rest)
			if !strings.HasPrefix(rest, "[") {
				index = listStart - 1
				continue
			}
			requirements, listEnd := pythonStringList(content, listStart)
			groups = append(groups, extrasGroup{name: name, requirements: requirements})
			index = listEnd
		}
	}
	return groups
}

// extrasGroup is one extras_require entry.
type extrasGroup struct {
	name         string
	requirements []string
}
