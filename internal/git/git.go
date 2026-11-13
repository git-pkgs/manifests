package git

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

const keyValueParts = 2 // key=value

func init() {
	core.Register("git", core.Manifest, &gitmodulesParser{}, core.ExactMatch(".gitmodules"))
}

// gitmodulesParser parses .gitmodules files.
type gitmodulesParser struct{}

var submoduleHeaderRegex = regexp.MustCompile(`^\[submodule\s+"([^"]+)"\]`)

type submodule struct {
	name   string
	path   string
	url    string
	branch string
}

func (p *gitmodulesParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	var current *submodule
	for i := 0; i < len(lines); i++ {
		physical := strings.TrimSuffix(lines[i], "\r")
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

		if current == nil {
			continue
		}

		key, raw, ok := parseKeyValue(physical)
		if !ok {
			continue
		}
		val, more, ok := configValue(raw)
		for more {
			if i+1 < len(lines) {
				i++
				raw += "\n" + strings.TrimSuffix(lines[i], "\r")
			} else {
				raw += "\n"
			}
			val, more, ok = configValue(raw)
		}
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

func parseKeyValue(line string) (key, value string, ok bool) {
	parts := strings.SplitN(line, "=", keyValueParts)
	if len(parts) != keyValueParts {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), parts[1], true
}

// configValue decodes a value as git's parse_value does; more means it continues on the next line, !ok that git rejects it.
func configValue(s string) (value string, more, ok bool) {
	var b strings.Builder
	quote := false
	trim := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !quote && strings.IndexByte(" \t\v\f\r", c) >= 0 {
			if trim == 0 {
				trim = b.Len()
			}
			if b.Len() > 0 {
				b.WriteByte(c)
			}
			continue
		}
		if !quote && (c == ';' || c == '#') {
			break
		}
		trim = 0
		switch c {
		case '\\':
			i++
			if i == len(s) {
				return "", true, true
			}
			switch s[i] {
			case '\n':
				continue
			case 't':
				b.WriteByte('\t')
			case 'b':
				b.WriteByte('\b')
			case 'n':
				b.WriteByte('\n')
			case '\\', '"':
				b.WriteByte(s[i])
			default:
				return "", false, false
			}
		case '"':
			quote = !quote
		default:
			b.WriteByte(c)
		}
	}
	if quote {
		return "", false, false
	}
	value = b.String()
	if trim > 0 {
		value = value[:trim]
	}
	return value, false, true
}
