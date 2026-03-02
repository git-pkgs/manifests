package guix

import (
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("guix", core.Manifest, &manifestParser{},
		core.AnyMatch(
			core.ExactMatch("manifest.scm"),
			core.SuffixMatch("-manifest.scm"),
		))
}

type manifestParser struct{}

// Match quoted strings inside specifications->manifest or specifications->manifest+.
var specStringRegex = regexp.MustCompile(`"([^"]+)"`)

func (p *manifestParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	text := string(content)

	// Find the specifications->manifest call
	idx := strings.Index(text, "specifications->manifest")
	if idx == -1 {
		return nil, nil
	}

	// Find the opening paren of the list argument
	rest := text[idx:]
	listStart := strings.Index(rest, "'(")
	if listStart == -1 {
		listStart = strings.Index(rest, "(list")
		if listStart == -1 {
			return nil, nil
		}
	}

	// Extract the balanced parenthesized list
	listText := rest[listStart:]
	depth := 0
	end := -1
	for i, ch := range listText {
		if ch == '(' {
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}
	if end == -1 {
		return nil, nil
	}

	block := listText[:end]

	// Strip ;; comments
	var filtered []string
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if commentIdx := strings.Index(trimmed, ";;"); commentIdx != -1 {
			trimmed = trimmed[:commentIdx]
		}
		filtered = append(filtered, trimmed)
	}
	block = strings.Join(filtered, "\n")

	var deps []core.Dependency
	for _, match := range specStringRegex.FindAllStringSubmatch(block, -1) {
		name := match[1]
		// Guix specs can include version: "package@version"
		version := ""
		if at := strings.Index(name, "@"); at != -1 {
			version = name[at+1:]
			name = name[:at]
		}
		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}
