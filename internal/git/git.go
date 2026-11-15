package git

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("git", core.Manifest, &gitmodulesParser{}, core.ExactMatch(".gitmodules"))
}

// gitmodulesParser parses .gitmodules files.
type gitmodulesParser struct{}

// Section names are case-insensitive in git config, so `[Submodule "..."]` is
// as valid as `[submodule "..."]`. The quoted subsection name is case-sensitive
// and is captured verbatim.
var submoduleHeaderRegex = regexp.MustCompile(`(?i)^\[submodule\s+"([^"]+)"\]`)

type submodule struct {
	name   string
	path   string
	url    string
	branch string
}

func (p *gitmodulesParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	s := string(content)

	var current *submodule
	for pos := 0; pos < len(s); {
		start := pos
		pos = lineEnd(s, start)
		physical := strings.TrimSuffix(strings.TrimSuffix(s[start:pos], "\n"), "\r")
		line := strings.TrimSpace(physical)
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if match := submoduleHeaderRegex.FindStringSubmatch(line); match != nil {
			if current != nil {
				deps = append(deps, current.dependency())
			}
			current = &submodule{name: match[1]}
			continue
		}

		// Any other section header ends the current submodule. Its keys belong
		// to that section, not to the submodule that came before it.
		if strings.HasPrefix(line, "[") {
			if current != nil {
				deps = append(deps, current.dependency())
				current = nil
			}
			continue
		}

		if current == nil {
			// A value continued with a backslash newline may continue with a
			// line that looks like a submodule header. Consume the whole value
			// so that line is not mistaken for the start of a submodule.
			if eq := strings.IndexByte(physical, '='); eq >= 0 {
				_, n, _ := configValue(s[start+eq+1:])
				pos = start + eq + 1 + n
			}
			continue
		}

		eq := strings.IndexByte(physical, '=')
		if eq < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(physical[:eq]))
		val, n, ok := configValue(s[start+eq+1:])
		pos = start + eq + 1 + n
		if !ok {
			continue
		}
		switch key {
		case "path":
			current.path = val
		case "url":
			current.url = val
		case "branch":
			current.branch = val
		}
	}

	if current != nil {
		deps = append(deps, current.dependency())
	}

	return &core.Result{Dependencies: deps}, nil
}

// dependency identifies the submodule by the host and path of its clone URL.
// The checkout path is where the superproject places it, not what it is, so
// a submodule whose URL is relative or local has no package identity.
func (s *submodule) dependency() core.Dependency {
	dep := core.Dependency{
		Name:   s.path,
		Scope:  core.Runtime,
		Direct: true,
	}
	if dep.Name == "" {
		dep.Name = s.name
	}
	if s.url != "" {
		dep.Source = core.Source{Kind: core.SourceGit, Value: s.url, Branch: s.branch}
	}
	if coordinate := core.GitCoordinate(s.url); coordinate != "" {
		dep.Name = coordinate
	} else {
		dep.NoPURL = true
	}
	return dep
}

// lineEnd returns the index just past the newline at or after i, or len(s).
func lineEnd(s string, i int) int {
	if j := strings.IndexByte(s[i:], '\n'); j >= 0 {
		return i + j + 1
	}
	return len(s)
}

// configValue decodes the value at the start of s as git's parse_value does, in one pass; n is the bytes used through the newline that ends it, !ok means git rejects it.
func configValue(s string) (value string, n int, ok bool) {
	var b strings.Builder
	quote, comment := false, false
	trim := 0
	for n < len(s) {
		c := s[n]
		n++
		if c == '\r' && n < len(s) && s[n] == '\n' {
			c = '\n'
			n++
		}
		if c == '\n' {
			break
		}
		if comment {
			continue
		}
		if !quote && (c == ' ' || c == '\t' || c == '\r') {
			if trim == 0 {
				trim = b.Len()
			}
			if b.Len() > 0 {
				b.WriteByte(c)
			}
			continue
		}
		if !quote && (c == ';' || c == '#') {
			comment = true
			continue
		}
		trim = 0
		switch c {
		case '\\':
			if n == len(s) {
				continue
			}
			e := s[n]
			n++
			if e == '\r' && n < len(s) && s[n] == '\n' {
				e = '\n'
				n++
			}
			switch e {
			case '\n':
				continue
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'n':
				b.WriteByte('\n')
			case '\\', '"':
				b.WriteByte(e)
			default:
				return "", lineEnd(s, n), false
			}
		case '"':
			quote = !quote
		default:
			b.WriteByte(c)
		}
	}
	if quote {
		return "", n, false
	}
	value = b.String()
	if trim > 0 {
		value = value[:trim]
	}
	return value, n, true
}
