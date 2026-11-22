package npm

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"net/url"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"go.yaml.in/yaml/v3"
)

func init() {
	core.Register("npm", core.Supplement, &pnpmWorkspaceParser{}, core.ExactMatch("pnpm-workspace.yaml"))
}

// jsonMember is one key and raw value of a JSON object, in source order.
type jsonMember struct {
	key   string
	value jsontext.Value
}

// jsonObjectMembers returns the members of a JSON object in source order, or
// nil when raw is not an object.
func jsonObjectMembers(raw []byte) []jsonMember {
	if len(raw) == 0 {
		return nil
	}
	decoder := jsontext.NewDecoder(bytes.NewReader(raw), json.DefaultOptionsV1())
	if !decodeJSONObjectOpening(decoder) {
		return nil
	}
	var members []jsonMember
	for decoder.PeekKind() != '}' {
		key, ok := decodeJSONKey(decoder)
		if !ok {
			return nil
		}
		value, err := decoder.ReadValue()
		if err != nil {
			return nil
		}
		members = append(members, jsonMember{key: key, value: value.Clone()})
	}
	return members
}

// jsonString decodes value when it is a JSON string.
func jsonString(value jsontext.Value) (string, bool) {
	if value.Kind() != '"' {
		return "", false
	}
	var text string
	if err := json.Unmarshal(value, &text); err != nil {
		return "", false
	}
	return text, true
}

// packageJSONReplacements collects override, resolution and patch entries
// from package.json, block by block and in source order within each block.
func packageJSONReplacements(overrides, resolutions, pnpm json.RawMessage) []core.Replacement {
	var replacements []core.Replacement
	locations := make(map[string]int)
	collectNpmOverrides(&replacements, locations, overrides)
	for _, member := range jsonObjectMembers(resolutions) {
		if spec, ok := jsonString(member.value); ok && !isNpmComment(member.key) {
			name, version := yarnResolutionTarget(member.key)
			appendNpmReplacement(&replacements, locations, "resolutions/"+url.PathEscape(member.key), name, version, spec)
		}
	}
	for _, member := range jsonObjectMembers(pnpm) {
		switch member.key {
		case "overrides":
			collectPnpmOverrides(&replacements, locations, "pnpm/overrides/", jsonStringMembers(member.value))
		case "patchedDependencies":
			collectPnpmPatches(&replacements, locations, "pnpm/patchedDependencies/", jsonStringMembers(member.value))
		}
	}
	return replacements
}

// collectNpmOverrides walks an npm overrides object in one pass over its
// tokens, so nested objects are read once however deep they go.
func collectNpmOverrides(replacements *[]core.Replacement, locations map[string]int, raw []byte) {
	if len(raw) == 0 {
		return
	}
	decoder := jsontext.NewDecoder(bytes.NewReader(raw), json.DefaultOptionsV1())
	if !decodeJSONObjectOpening(decoder) {
		return
	}
	walker := npmOverrideWalker{
		decoder:      decoder,
		replacements: replacements,
		locations:    locations,
		path:         []byte("overrides/"),
	}
	walker.object("", 0)
}

// maxNpmOverrideDepth bounds how many objects deep an overrides walk goes.
// Real overrides nest a few levels, while every recorded Location carries
// the full path, so deeper objects are skipped.
const maxNpmOverrideDepth = 32

// npmOverrideWalker holds the decoder and the location path shared by every
// level of an overrides walk. The path is extended and truncated in place.
type npmOverrideWalker struct {
	decoder      *jsontext.Decoder
	replacements *[]core.Replacement
	locations    map[string]int
	path         []byte
}

// object reads the members of an open object, depth objects below overrides,
// through its closing brace. A nested object scopes its entries to parent,
// the package named by its key. Its "." key overrides that parent.
func (w *npmOverrideWalker) object(parent string, depth int) bool {
	for w.decoder.PeekKind() != '}' {
		key, ok := decodeJSONKey(w.decoder)
		if !ok {
			return false
		}
		kind := w.decoder.PeekKind()
		tooDeep := kind == '{' && depth >= maxNpmOverrideDepth
		if isNpmComment(key) || tooDeep || (kind != '"' && kind != '{') {
			if !skipJSONValue(w.decoder) {
				return false
			}
			continue
		}
		token, err := w.decoder.ReadToken()
		if err != nil {
			return false
		}
		if kind == '{' {
			mark := len(w.path)
			w.path = append(append(w.path, url.PathEscape(key)...), '/')
			if !w.object(key, depth+1) {
				return false
			}
			w.path = w.path[:mark]
			continue
		}
		target := key
		if key == "." {
			if parent == "" {
				continue
			}
			target = parent
		}
		name, version := parseNPMSpec(target)
		mark := len(w.path)
		w.path = append(w.path, url.PathEscape(key)...)
		appendNpmReplacement(w.replacements, w.locations, string(w.path), name, version, token.String())
		w.path = w.path[:mark]
	}
	_, err := w.decoder.ReadToken()
	return err == nil
}

// jsonStringMembers returns the string-valued members of a JSON object.
func jsonStringMembers(raw []byte) []stringMember {
	var members []stringMember
	for _, member := range jsonObjectMembers(raw) {
		if value, ok := jsonString(member.value); ok && !isNpmComment(member.key) {
			members = append(members, stringMember{key: member.key, value: value})
		}
	}
	return members
}

// stringMember is one key and string value of a mapping, in source order.
type stringMember struct {
	key   string
	value string
}

// collectPnpmOverrides records pnpm overrides, whose keys may name a parent
// chain such as "parent@1>child@2".
func collectPnpmOverrides(replacements *[]core.Replacement, locations map[string]int, prefix string, members []stringMember) {
	for _, member := range members {
		name, version := parseNPMSpec(pnpmOverrideTarget(member.key))
		appendNpmReplacement(replacements, locations, prefix+url.PathEscape(member.key), name, version, member.value)
	}
}

// pnpmOverrideTarget returns the target of a pnpm override selector. Like
// pnpm, it splits at the first ">" not preceded by a space, "|" or "@", so a
// comparison such as "foo@>2" is a range rather than a parent delimiter.
func pnpmOverrideTarget(selector string) string {
	for index := 1; index < len(selector); index++ {
		if selector[index] == '>' && !strings.ContainsRune(" |@", rune(selector[index-1])) {
			return selector[index+1:]
		}
	}
	return selector
}

// collectPnpmPatches records pnpm patchedDependencies, keyed by name or
// name@version and pointing at a patch file.
func collectPnpmPatches(replacements *[]core.Replacement, locations map[string]int, prefix string, members []stringMember) {
	for _, member := range members {
		name, version := parseNPMSpec(member.key)
		if name == "" {
			continue
		}
		*replacements = append(*replacements, core.Replacement{
			Name:     name,
			Version:  version,
			Patch:    member.value,
			Location: core.NextLocation(locations, prefix+url.PathEscape(member.key)),
		})
	}
}

func appendNpmReplacement(
	replacements *[]core.Replacement,
	locations map[string]int,
	location, name, version, spec string,
) {
	if name == "" {
		return
	}
	replacement := npmReplacementSpec(spec)
	// A patch of the target itself is not a package swap.
	if replacement.Patch != "" && replacement.ReplacementName == name {
		replacement.ReplacementName = ""
	}
	replacement.Name = name
	replacement.Version = version
	replacement.Location = core.NextLocation(locations, location)
	*replacements = append(*replacements, replacement)
}

// yarnResolutionTarget returns the package a Yarn resolution key targets: the
// last package in a path such as "parent/**/@scope/child@^1".
func yarnResolutionTarget(key string) (string, string) {
	segments := strings.Split(key, "/")
	last := len(segments) - 1
	target := segments[last]
	if last > 0 && strings.HasPrefix(segments[last-1], "@") {
		target = segments[last-1] + "/" + target
	}
	if target == "**" || target == "*" {
		return "", ""
	}
	return parseNPMSpec(target)
}

// npmReplacementSpec classifies the value of an override or resolution,
// keeping it as written. Values that are not a package alias, source or
// patch, including "$name" references, stay in ReplacementVersion.
func npmReplacementSpec(spec string) core.Replacement {
	switch {
	case strings.HasPrefix(spec, "npm:"):
		reference := strings.TrimPrefix(spec, "npm:")
		if isNpmRange(reference) {
			return core.Replacement{ReplacementVersion: reference}
		}
		name, version := parseNPMSpec(reference)
		return core.Replacement{ReplacementName: name, ReplacementVersion: version}
	case strings.HasPrefix(spec, "patch:"):
		return yarnPatchReplacement(strings.TrimPrefix(spec, "patch:"))
	case strings.HasPrefix(spec, "github:"):
		value, ref, _ := strings.Cut(strings.TrimPrefix(spec, "github:"), "#")
		return core.Replacement{Source: core.Source{Kind: core.SourceGitHub, Value: value, Ref: ref}}
	case strings.HasPrefix(spec, "git+"), strings.HasPrefix(spec, "git://"), strings.HasPrefix(spec, "git@"):
		value, ref, _ := strings.Cut(spec, "#")
		return core.Replacement{Source: core.Source{Kind: core.SourceGit, Value: value, Ref: ref}}
	case strings.HasPrefix(spec, "http://"), strings.HasPrefix(spec, "https://"):
		return core.Replacement{Source: core.Source{Kind: core.SourceURL, Value: spec}}
	case strings.HasPrefix(spec, "file:"), strings.HasPrefix(spec, "link:"), strings.HasPrefix(spec, "portal:"),
		strings.HasPrefix(spec, "./"), strings.HasPrefix(spec, "../"), strings.HasPrefix(spec, "/"):
		return core.Replacement{Source: core.Source{Kind: core.SourcePath, Value: spec}}
	case !strings.Contains(spec, ":") && strings.Contains(spec, "/") && !strings.HasPrefix(spec, "@"):
		value, ref, _ := strings.Cut(spec, "#")
		return core.Replacement{Source: core.Source{Kind: core.SourceGitHub, Value: value, Ref: ref}}
	}
	return core.Replacement{ReplacementVersion: spec}
}

// npmComparator matches one comparator of a semver range: a full version,
// which may carry prerelease and build parts, or a partial or X-range.
var npmComparator = regexp.MustCompile(`^(?:[<>]=?|=|\^|~)?v?(?:\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?|(?:\d+|[xX*])(?:\.(?:\d+|[xX*])){0,2})$`)

// npmOperator matches a comparison operator written apart from its version.
var npmOperator = regexp.MustCompile(`^(?:[<>]=?|=|\^|~)$`)

// isNpmRange reports whether an npm: reference is a range for the same
// package, as Yarn accepts in "npm:4.17.21", rather than an aliased name.
// Package names may start with a digit, so the whole reference must parse
// as a range.
func isNpmRange(reference string) bool {
	tokens := strings.Fields(strings.ReplaceAll(reference, "||", " "))
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if token != "-" && !npmOperator.MatchString(token) && !npmComparator.MatchString(token) {
			return false
		}
	}
	return true
}

// yarnPatchReplacement reads a Yarn patch: descriptor such as
// "lodash@npm%3A4.17.21#./.yarn/patches/lodash.patch::locator=app".
func yarnPatchReplacement(descriptor string) core.Replacement {
	source, patch, _ := strings.Cut(descriptor, "#")
	patch, _, _ = strings.Cut(patch, "::")
	if decoded, err := url.PathUnescape(source); err == nil {
		source = decoded
	}
	name, version := parseNPMSpec(source)
	version = strings.TrimPrefix(version, "npm:")
	return core.Replacement{ReplacementName: name, ReplacementVersion: version, Patch: patch}
}

// pnpmWorkspaceParser reads the overrides and patchedDependencies that pnpm 10
// keeps in pnpm-workspace.yaml.
type pnpmWorkspaceParser struct{}

func (p *pnpmWorkspaceParser) Parse(filename string, content []byte) (*core.Result, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}
	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return &core.Result{}, nil
	}
	var replacements []core.Replacement
	locations := make(map[string]int)
	root := document.Content[0]
	for index := 0; index+1 < len(root.Content); index += 2 {
		switch root.Content[index].Value {
		case "overrides":
			collectPnpmOverrides(&replacements, locations, "overrides/", yamlStringMembers(root.Content[index+1]))
		case "patchedDependencies":
			collectPnpmPatches(&replacements, locations, "patchedDependencies/", yamlStringMembers(root.Content[index+1]))
		}
	}
	return &core.Result{Replacements: replacements}, nil
}

// yamlStringMembers returns the scalar-valued entries of a YAML mapping.
func yamlStringMembers(node *yaml.Node) []stringMember {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	var members []stringMember
	for index := 0; index+1 < len(node.Content); index += 2 {
		key, value := node.Content[index], node.Content[index+1]
		if key.Kind == yaml.ScalarNode && value.Kind == yaml.ScalarNode {
			members = append(members, stringMember{key: key.Value, value: value.Value})
		}
	}
	return members
}
