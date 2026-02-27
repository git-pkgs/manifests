package ips

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("ips", core.Manifest, &p5mParser{}, core.SuffixMatch(".p5m"))
}

type p5mParser struct{}

func (p *p5mParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	deps := make([]core.Dependency, 0, core.EstimateDeps(len(content)))
	text := string(content)

	// Join continuation lines (backslash-newline) into single lines.
	var lines []string
	var buf strings.Builder
	core.ForEachLine(text, func(line string) bool {
		trimmed := strings.TrimRight(line, " \t\r")
		if before, ok := strings.CutSuffix(trimmed, "\\"); ok {
			buf.WriteString(before)
			buf.WriteByte(' ')
		} else {
			if buf.Len() > 0 {
				buf.WriteString(trimmed)
				lines = append(lines, buf.String())
				buf.Reset()
			} else {
				lines = append(lines, trimmed)
			}
		}
		return true
	})
	if buf.Len() > 0 {
		lines = append(lines, buf.String())
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "depend ") {
			continue
		}

		// Skip macro references and TBD placeholders.
		if strings.Contains(line, "$(") || strings.Contains(line, "fmri=__TBD") {
			continue
		}

		depType := extractAttr(line, "type=")
		if depType == "" {
			continue
		}

		scope := mapScope(depType)

		// require-any can have multiple fmri= attributes.
		fmris := extractAllAttrs(line, "fmri=")
		for _, fmri := range fmris {
			name, version := parseFMRI(fmri)
			if name == "" {
				continue
			}
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   scope,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// parseFMRI extracts name and version from an IPS FMRI like
// "pkg:/library/libxml2@2.9.14,5.11-2024.0.0.0".
func parseFMRI(fmri string) (name, version string) {
	// Strip pkg:/ or pkg:// prefix.
	s := fmri
	if strings.HasPrefix(s, "pkg://") {
		s = s[len("pkg://"):]
		// Skip publisher up to the first /.
		if idx := strings.IndexByte(s, '/'); idx >= 0 {
			s = s[idx+1:]
		}
	} else if strings.HasPrefix(s, "pkg:/") {
		s = s[len("pkg:/"):]
	}

	// Split on @ for name and version.
	if before, after, ok := strings.Cut(s, "@"); ok {
		name = before
		version, _, _ = strings.Cut(after, ",")
	} else {
		name = s
	}
	return name, version
}

func mapScope(depType string) core.Scope {
	switch depType {
	case "require", "require-any", "group", "incorporate":
		return core.Runtime
	case "optional", "conditional":
		return core.Optional
	default:
		return core.Runtime
	}
}

// extractAttr returns the value of the first occurrence of key (e.g. "type=")
// in the space-delimited attribute line.
func extractAttr(line, key string) string {
	_, after, ok := strings.Cut(line, key)
	if !ok {
		return ""
	}
	if val, _, ok := strings.Cut(after, " "); ok {
		return val
	}
	return after
}

// extractAllAttrs returns all values for a repeated attribute key.
func extractAllAttrs(line, key string) []string {
	var vals []string
	rest := line
	for {
		_, after, ok := strings.Cut(rest, key)
		if !ok {
			break
		}
		if val, remainder, ok := strings.Cut(after, " "); ok {
			vals = append(vals, val)
			rest = remainder
		} else {
			vals = append(vals, after)
			break
		}
	}
	return vals
}
