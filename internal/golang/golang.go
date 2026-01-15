package golang

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	// go.mod - manifest
	core.Register("golang", core.Manifest, &goModParser{}, core.ExactMatch("go.mod"))

	// go.sum - lockfile
	core.Register("golang", core.Lockfile, &goSumParser{}, core.ExactMatch("go.sum"))
}

// goModParser parses go.mod files.
type goModParser struct{}

var (
	// Single-line require: require example.com/pkg v1.2.3
	singleRequireRegex = regexp.MustCompile(`^\s*require\s+(\S+)\s+(\S+)`)

	// Multi-line require block entry: example.com/pkg v1.2.3 // indirect
	requireEntryRegex = regexp.MustCompile(`^\s*(\S+)\s+(\S+)(?:\s*//.*)?$`)
)

func (p *goModParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	inRequireBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		// Detect require block start
		if strings.HasPrefix(trimmed, "require (") || trimmed == "require (" {
			inRequireBlock = true
			continue
		}

		// Detect require block end
		if inRequireBlock && trimmed == ")" {
			inRequireBlock = false
			continue
		}

		// Single-line require
		if strings.HasPrefix(trimmed, "require ") && !strings.Contains(trimmed, "(") {
			if match := singleRequireRegex.FindStringSubmatch(trimmed); match != nil {
				direct := !strings.Contains(line, "// indirect")
				deps = append(deps, core.Dependency{
					Name:    match[1],
					Version: match[2],
					Scope:   core.Runtime,
					Direct:  direct,
				})
			}
			continue
		}

		// Inside require block
		if inRequireBlock {
			if match := requireEntryRegex.FindStringSubmatch(trimmed); match != nil {
				direct := !strings.Contains(line, "// indirect")
				deps = append(deps, core.Dependency{
					Name:    match[1],
					Version: match[2],
					Scope:   core.Runtime,
					Direct:  direct,
				})
			}
		}
	}

	return deps, nil
}

// goSumParser parses go.sum files.
type goSumParser struct{}

var (
	// go.sum line: module/path v1.2.3 h1:hash=
	goSumLineRegex = regexp.MustCompile(`^(\S+)\s+(\S+)\s+(h1:\S+)$`)
)

func (p *goSumParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[string]bool)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if match := goSumLineRegex.FindStringSubmatch(trimmed); match != nil {
			name := match[1]
			version := match[2]

			// Skip /go.mod entries, only keep actual module checksums
			if strings.HasSuffix(version, "/go.mod") {
				continue
			}

			// Deduplicate (go.sum can have multiple entries per module)
			key := name + "@" + version
			if seen[key] {
				continue
			}
			seen[key] = true

			deps = append(deps, core.Dependency{
				Name:      name,
				Version:   version,
				Scope:   core.Runtime,
				Integrity: match[3],
				Direct:    false, // go.sum doesn't track direct vs indirect
			})
		}
	}

	return deps, nil
}
