package npm

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"strings"
)

func init() {
	core.Register("deno", core.Manifest, &denoJSONParser{}, core.ExactMatch("deno.json"))
	core.Register("deno", core.Manifest, &denoJSONParser{}, core.ExactMatch("deno.jsonc"))
	core.Register("deno", core.Lockfile, &denoLockParser{}, core.ExactMatch("deno.lock"))
}

// denoJSONParser parses deno.json files.
type denoJSONParser struct{}

type denoJSON struct {
	Imports map[string]string `json:"imports"`
}

func (p *denoJSONParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deno denoJSON
	if err := json.Unmarshal(content, &deno); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	for _, spec := range deno.Imports {
		name, version := parseDenoSpec(spec)
		if name != "" {
			deps = append(deps, core.Dependency{
				Name:    name,
				Version: version,
				Scope:   core.Runtime,
				Direct:  true,
			})
		}
	}

	return deps, nil
}

// denoLockParser parses deno.lock files.
type denoLockParser struct{}

type denoLock struct {
	Specifiers map[string]string       `json:"specifiers"`
	JSR        map[string]denoLockPkg  `json:"jsr"`
	NPM        map[string]denoLockPkg  `json:"npm"`
}

type denoLockPkg struct {
	Integrity string `json:"integrity"`
}

func (p *denoLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var lock denoLock
	if err := json.Unmarshal(content, &lock); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency

	// Parse JSR packages
	for pkgVer, pkg := range lock.JSR {
		name, version := parseDenoLockPkg(pkgVer)
		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:   core.Runtime,
			Integrity: pkg.Integrity,
			Direct:    false,
		})
	}

	// Parse NPM packages
	for pkgVer, pkg := range lock.NPM {
		name, version := parseDenoLockPkg(pkgVer)
		deps = append(deps, core.Dependency{
			Name:      name,
			Version:   version,
			Scope:   core.Runtime,
			Integrity: pkg.Integrity,
			Direct:    false,
		})
	}

	return deps, nil
}

// parseDenoSpec parses a deno import spec like "npm:chalk@5.3.0" or "jsr:@std/path@^1.0.0".
func parseDenoSpec(spec string) (name, version string) {
	// Handle npm: prefix
	if strings.HasPrefix(spec, "npm:") {
		spec = strings.TrimPrefix(spec, "npm:")
		return parseNPMSpec(spec)
	}

	// Handle jsr: prefix
	if strings.HasPrefix(spec, "jsr:") {
		spec = strings.TrimPrefix(spec, "jsr:")
		return parseJSRSpec(spec)
	}

	// Handle https: URLs (skip for now)
	if strings.HasPrefix(spec, "https:") {
		return "", ""
	}

	return spec, ""
}

// parseNPMSpec parses an npm spec like "chalk@5.3.0" or "chalk".
func parseNPMSpec(spec string) (name, version string) {
	// Handle scoped packages
	if strings.HasPrefix(spec, "@") {
		if idx := strings.Index(spec[1:], "@"); idx > 0 {
			return spec[:idx+1], spec[idx+2:]
		}
		return spec, ""
	}

	// Handle regular packages
	if idx := strings.Index(spec, "@"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	return spec, ""
}

// parseJSRSpec parses a JSR spec like "@std/path@^1.0.0" or "@std/path".
func parseJSRSpec(spec string) (name, version string) {
	// JSR packages are always scoped like @scope/name@version
	if idx := strings.LastIndex(spec, "@"); idx > 0 {
		return spec[:idx], spec[idx+1:]
	}
	return spec, ""
}

// parseDenoLockPkg parses a lock package key like "@std/fs@1.0.3" or "chalk@5.3.0".
func parseDenoLockPkg(key string) (name, version string) {
	// Handle scoped packages
	if strings.HasPrefix(key, "@") {
		if idx := strings.LastIndex(key, "@"); idx > 0 {
			return key[:idx], key[idx+1:]
		}
		return key, ""
	}

	// Handle regular packages
	if idx := strings.Index(key, "@"); idx > 0 {
		return key[:idx], key[idx+1:]
	}
	return key, ""
}
