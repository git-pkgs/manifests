package golang

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	// go.mod - manifest
	core.Register("golang", core.Manifest, &goModParser{}, core.ExactMatch("go.mod"))

	// go.sum - supplement (provides integrity hashes for go.mod dependencies)
	core.Register("golang", core.Supplement, &goSumParser{}, core.ExactMatch("go.sum"))

	// go.graph - lockfile (go mod graph output)
	core.Register("golang", core.Lockfile, &goGraphParser{}, core.ExactMatch("go.graph"))
}

// goModParser parses go.mod files.
type goModParser struct{}

type moduleVersion struct {
	path    string
	version string
}

var (
	// Single-line require: require example.com/pkg v1.2.3
	singleRequireRegex = regexp.MustCompile(`^\s*require\s+(\S+)\s+(\S+)`)

	// Multi-line require block entry: example.com/pkg v1.2.3 // indirect
	requireEntryRegex = regexp.MustCompile(`^\s*(\S+)\s+(\S+)(?:\s*//.*)?$`)

	// Single-line tool: tool example.com/pkg/cmd/foo
	singleToolRegex = regexp.MustCompile(`^\s*tool\s+(\S+)`)

	// Multi-line tool block entry: example.com/pkg/cmd/foo
	toolEntryRegex = regexp.MustCompile(`^\s*(\S+)\s*$`)
)

func (p *goModParser) Parse(filename string, content []byte) (*core.Result, error) {
	lines := strings.Split(string(content), "\n")
	tools := collectToolPaths(lines)
	replaced := collectReplacedModules(lines)
	deps, declarations := collectRequireDeps(lines, tools, replaced)

	var modulePath string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") || strings.HasPrefix(trimmed, "module\t") {
			modulePath = strings.TrimSpace(strings.Trim(strings.TrimSpace(trimmed[len("module"):]), `"`))
			break
		}
	}

	return &core.Result{Name: modulePath, Dependencies: deps, Declarations: declarations}, nil
}

// collectToolPaths scans go.mod lines for tool directives (both single-line and block form)
// and returns a set of tool import paths.
func collectToolPaths(lines []string) map[string]bool {
	tools := make(map[string]bool)
	inToolBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		if strings.HasPrefix(trimmed, "tool (") || trimmed == "tool (" {
			inToolBlock = true
			continue
		}

		if inToolBlock && trimmed == ")" {
			inToolBlock = false
			continue
		}

		if strings.HasPrefix(trimmed, "tool ") && !strings.Contains(trimmed, "(") {
			if match := singleToolRegex.FindStringSubmatch(trimmed); match != nil {
				tools[match[1]] = true
			}
			continue
		}

		if inToolBlock {
			if match := toolEntryRegex.FindStringSubmatch(trimmed); match != nil {
				tools[match[1]] = true
			}
		}
	}

	return tools
}

// collectRequireDeps scans go.mod lines for require directives (both single-line and block form)
// and returns dependencies, marking tool-related modules as development scope.
func collectRequireDeps(lines []string, tools map[string]bool, replaced map[moduleVersion]bool) ([]core.Dependency, []core.Declaration) {
	var deps []core.Dependency
	var declarations []core.Declaration
	inRequireBlock := false
	locations := make(map[string]int)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}

		if strings.HasPrefix(trimmed, "require (") || trimmed == "require (" {
			inRequireBlock = true
			continue
		}

		if inRequireBlock && trimmed == ")" {
			inRequireBlock = false
			continue
		}

		if strings.HasPrefix(trimmed, "require ") && !strings.Contains(trimmed, "(") {
			if match := singleRequireRegex.FindStringSubmatch(trimmed); match != nil {
				dep := newRequireDep(match[1], match[2], line, tools)
				deps = append(deps, dep)
				appendGoDeclaration(&declarations, locations, dep, replaced)
			}
			continue
		}

		if inRequireBlock {
			if match := requireEntryRegex.FindStringSubmatch(trimmed); match != nil {
				dep := newRequireDep(match[1], match[2], line, tools)
				deps = append(deps, dep)
				appendGoDeclaration(&declarations, locations, dep, replaced)
			}
		}
	}

	return deps, declarations
}

// appendGoDeclaration records a require directive unless a replace directive
// changes that module's source.
func appendGoDeclaration(
	declarations *[]core.Declaration,
	locations map[string]int,
	dependency core.Dependency,
	replaced map[moduleVersion]bool,
) {
	if replaced[moduleVersion{path: dependency.Name}] ||
		replaced[moduleVersion{path: dependency.Name, version: dependency.Version}] {
		return
	}
	location := core.NextLocation(locations, "require/"+url.PathEscape(dependency.Name))
	*declarations = append(*declarations, core.Declaration{
		Name:     dependency.Name,
		Version:  dependency.Version,
		Scope:    dependency.Scope,
		Direct:   dependency.Direct,
		Location: location,
	})
}

// collectReplacedModules returns module paths named on the left side of a
// replace directive, in either single-line or block form.
func collectReplacedModules(lines []string) map[moduleVersion]bool {
	replaced := make(map[moduleVersion]bool)
	inReplaceBlock := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if strings.HasPrefix(trimmed, "replace (") {
			inReplaceBlock = true
			continue
		}
		if inReplaceBlock && trimmed == ")" {
			inReplaceBlock = false
			continue
		}

		spec := ""
		if strings.HasPrefix(trimmed, "replace ") && !strings.Contains(trimmed, "(") {
			spec = strings.TrimSpace(strings.TrimPrefix(trimmed, "replace "))
		} else if inReplaceBlock {
			spec = trimmed
		}
		left, _, ok := strings.Cut(spec, "=>")
		if !ok {
			continue
		}
		fields := strings.Fields(left)
		if len(fields) > 0 {
			module := moduleVersion{path: fields[0]}
			if len(fields) > 1 {
				module.version = fields[1]
			}
			replaced[module] = true
		}
	}
	return replaced
}

// newRequireDep builds a Dependency from a parsed require entry, determining
// scope based on whether the module is used by a tool directive.
func newRequireDep(name, version, rawLine string, tools map[string]bool) core.Dependency {
	direct := !strings.Contains(rawLine, "// indirect")
	scope := core.Runtime
	if isToolModule(name, tools) {
		scope = core.Development
	}
	return core.Dependency{
		Name:    name,
		Version: version,
		Scope:   scope,
		Direct:  direct,
	}
}

// isToolModule checks if a module is used by any tool.
// A module matches if it equals a tool path or if a tool path starts with the module path.
func isToolModule(module string, tools map[string]bool) bool {
	if tools[module] {
		return true
	}
	// Check if any tool path starts with this module
	// e.g., module "golang.org/x/tools" matches tool "golang.org/x/tools/cmd/stringer"
	for tool := range tools {
		if strings.HasPrefix(tool, module+"/") {
			return true
		}
	}
	return false
}

// goSumParser parses go.sum files.
type goSumParser struct{}

type goSumKey struct {
	name    string
	version string
}

func (p *goSumParser) Parse(filename string, content []byte) (*core.Result, error) {
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

	return &core.Result{Dependencies: deps}, nil
}

// goGraphParser parses go.graph files (go mod graph output).
type goGraphParser struct{}

func (p *goGraphParser) Parse(filename string, content []byte) (*core.Result, error) {
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

		const graphEdgeParts = 2 // parent dep
		parts := strings.Fields(line)
		if len(parts) != graphEdgeParts {
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

		const graphEdgeParts = 2 // parent dep
		parts := strings.Fields(line)
		if len(parts) != graphEdgeParts {
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

	return &core.Result{Dependencies: deps}, nil
}
