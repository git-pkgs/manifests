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

	// go.graph - lockfile (go mod graph output)
	core.Register("golang", core.Lockfile, &goGraphParser{}, core.ExactMatch("go.graph"))
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

type goSumKey struct {
	name    string
	version string
}

func (p *goSumParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[goSumKey]bool)
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse go.sum line: module/path v1.2.3 h1:hash=
		// Fast path: use string operations instead of regex
		sp1 := strings.IndexByte(line, ' ')
		if sp1 < 0 {
			continue
		}
		name := line[:sp1]

		rest := line[sp1+1:]
		sp2 := strings.IndexByte(rest, ' ')
		if sp2 < 0 {
			continue
		}
		version := rest[:sp2]
		hash := rest[sp2+1:]

		// Skip /go.mod entries, only keep actual module checksums
		if strings.HasSuffix(version, "/go.mod") {
			continue
		}

		// Only accept h1: hashes
		if !strings.HasPrefix(hash, "h1:") {
			continue
		}

		// Deduplicate (go.sum can have multiple entries per module)
		key := goSumKey{name, version}
		if seen[key] {
			continue
		}
		seen[key] = true

		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:     core.Runtime,
			Integrity: hash,
			Direct:    false, // go.sum doesn't track direct vs indirect
		})
	}

	return deps, nil
}

// goGraphParser parses go.graph files (go mod graph output).
type goGraphParser struct{}

func (p *goGraphParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	seen := make(map[string]bool)
	directDeps := make(map[string]bool)
	lines := strings.Split(string(content), "\n")

	// First pass: identify direct dependencies (those required by the main module)
	// The main module appears without a version in the first column
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		parent := parts[0]
		dep := parts[1]

		// If parent has no @version, it's the main module
		if !strings.Contains(parent, "@") {
			// Extract just the name from dep (before @)
			if idx := strings.LastIndex(dep, "@"); idx > 0 {
				directDeps[dep[:idx]] = true
			}
		}
	}

	// Second pass: collect all dependencies
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		dep := parts[1]

		// Parse name@version
		idx := strings.LastIndex(dep, "@")
		if idx <= 0 {
			continue
		}

		name := dep[:idx]
		version := dep[idx+1:]

		if seen[name] {
			continue
		}
		seen[name] = true

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  directDeps[name],
		})
	}

	return deps, nil
}
