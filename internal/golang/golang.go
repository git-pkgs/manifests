package golang

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	// go.mod - manifest
	core.Register("golang", core.Manifest, &goModParser{}, core.ExactMatch("go.mod"))

	// go.sum - supplement (provides integrity hashes for go.mod dependencies)
	core.Register("golang", core.Supplement, &goSumParser{}, core.ExactMatch("go.sum"))

	// go mod graph output saved under any of its common names
	core.Register("golang", core.Lockfile, &goGraphParser{}, core.ExactMatch("go.graph", "go.mod.graph", "go.modgraph"))
}

// goModParser parses go.mod files.
type goModParser struct{}

type moduleVersion struct {
	path    string
	version string
}

var (
	// Single-line tool: tool example.com/pkg/cmd/foo
	singleToolRegex = regexp.MustCompile(`^\s*tool\s+(\S+)`)

	// Multi-line tool block entry: example.com/pkg/cmd/foo
	toolEntryRegex = regexp.MustCompile(`^\s*(\S+)\s*$`)
)

func (p *goModParser) Parse(filename string, content []byte) (*core.Result, error) {
	text := string(content)
	tools := collectToolPaths(text)
	replacements := collectReplacements(text)
	deps, declarations := collectRequireDeps(text, tools, replacedModules(replacements))

	var modulePath string
	for line := range strings.SplitSeq(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") || strings.HasPrefix(trimmed, "module\t") {
			modulePath = strings.TrimSpace(strings.Trim(strings.TrimSpace(trimmed[len("module"):]), `"`))
			break
		}
	}

	return &core.Result{Name: modulePath, Dependencies: deps, Declarations: declarations, Replacements: replacements}, nil
}

// collectToolPaths scans go.mod lines for tool directives (both single-line and block form)
// and returns a set of tool import paths.
func collectToolPaths(text string) map[string]bool {
	tools := make(map[string]bool)
	inToolBlock := false

	for line := range strings.SplitSeq(text, "\n") {
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
func collectRequireDeps(text string, tools map[string]bool, replaced map[moduleVersion]bool) ([]core.Dependency, []core.Declaration) {
	var deps []core.Dependency
	var declarations []core.Declaration
	locations := make(map[string]int)
	for _, entry := range goModDirectives(text, "require") {
		const requireFields = 2 // path version
		if len(entry.tokens) < requireFields {
			continue
		}
		dep := newRequireDep(entry.tokens[0], entry.tokens[1], entry.line, tools)
		deps = append(deps, dep)
		appendGoDeclaration(&declarations, locations, dep, replaced)
	}
	return deps, declarations
}

// goModDirective is one single-line directive or block entry, with the verb
// removed from its tokens.
type goModDirective struct {
	tokens []string
	line   string
}

// goModDirectives returns the entries of every verb directive in either
// single-line or block form.
func goModDirectives(text, verb string) []goModDirective {
	var entries []goModDirective
	inBlock := false
	for line := range strings.SplitSeq(text, "\n") {
		tokens := goModTokens(line)
		switch {
		case len(tokens) == 0:
		case inBlock && tokens[0] == ")":
			inBlock = false
		case inBlock:
			entries = append(entries, goModDirective{tokens: tokens, line: line})
		case tokens[0] != verb:
		case len(tokens) == 2 && tokens[1] == "(":
			inBlock = true
		case len(tokens) == 3 && tokens[1] == "(" && tokens[2] == ")":
			// An empty block on one line has no entries.
		default:
			entries = append(entries, goModDirective{tokens: tokens[1:], line: line})
		}
	}
	return entries
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

// collectReplacements records every replace directive in source order,
// without applying it, since Go only honours replace in the main module.
func collectReplacements(text string) []core.Replacement {
	var replacements []core.Replacement
	locations := make(map[string]int)
	for _, entry := range goModDirectives(text, "replace") {
		arrow := slices.Index(entry.tokens, "=>")
		if arrow < 1 || arrow == len(entry.tokens)-1 {
			continue
		}
		replacement := core.Replacement{Name: entry.tokens[0]}
		if arrow > 1 {
			replacement.Version = entry.tokens[1]
		}
		right := entry.tokens[arrow+1:]
		if isLocalReplacement(right[0]) {
			replacement.Source = core.Source{Kind: core.SourcePath, Value: right[0]}
		} else {
			replacement.ReplacementName = right[0]
			if len(right) > 1 {
				replacement.ReplacementVersion = right[1]
			}
		}
		replacement.Location = core.NextLocation(locations, "replace/"+url.PathEscape(replacement.Name))
		replacements = append(replacements, replacement)
	}
	return replacements
}

// replacedModules returns the left side of each replacement.
func replacedModules(replacements []core.Replacement) map[moduleVersion]bool {
	replaced := make(map[moduleVersion]bool, len(replacements))
	for _, replacement := range replacements {
		replaced[moduleVersion{path: replacement.Name, version: replacement.Version}] = true
	}
	return replaced
}

// goModTokens splits a go.mod line into tokens as the go command does: tokens
// are separated by white space, unquoted parentheses are tokens of their own,
// double-quoted strings are decoded and "//" outside a quoted string begins a
// comment, even directly after a token.
func goModTokens(line string) []string {
	var tokens []string
	for {
		line = strings.TrimLeft(line, " \t\r")
		if line == "" || strings.HasPrefix(line, "//") {
			return tokens
		}
		quoted := line[0] == '"'
		var end int
		if quoted {
			end = quotedStringEnd(line)
		} else {
			end = unquotedTokenEnd(line)
		}
		if end < 0 {
			end = len(line)
		}
		token := line[:end]
		if quoted {
			if unquoted, err := strconv.Unquote(token); err == nil {
				token = unquoted
			}
		}
		tokens = append(tokens, token)
		line = line[end:]
	}
}

// unquotedTokenEnd returns the length of the unquoted token that starts s,
// stopping at white space, a parenthesis or "//". A leading parenthesis is a
// one-byte token. It stops at the token end, so tokenizing stays linear.
func unquotedTokenEnd(s string) int {
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '(' || c == ')':
			return max(i, 1)
		case c == ' ' || c == '\t' || c == '\r':
			return i
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			return i
		}
	}
	return len(s)
}

// quotedStringEnd returns the index just past the closing quote of the
// double-quoted string that starts s, or -1 when it is unterminated.
func quotedStringEnd(s string) int {
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i + 1
		}
	}
	return -1
}

// isLocalReplacement reports whether a replacement path names a directory.
// Like modfile.IsDirectoryPath, it accepts Unix and Windows path forms on any
// host because go.mod files move between systems.
func isLocalReplacement(path string) bool {
	return path == "." || path == ".." ||
		strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") ||
		strings.HasPrefix(path, `.\`) || strings.HasPrefix(path, `..\`) ||
		strings.HasPrefix(path, "/") || strings.HasPrefix(path, `\`) ||
		len(path) >= 2 && isASCIILetter(path[0]) && path[1] == ':'
}

func isASCIILetter(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
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
	text := string(content)

	for line := range strings.SplitSeq(text, "\n") {
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
	text := string(content)

	// First pass: identify direct dependencies (those required by the main module)
	// The main module appears without a version in the first column
	for line := range strings.SplitSeq(text, "\n") {
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
	for line := range strings.SplitSeq(text, "\n") {
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
