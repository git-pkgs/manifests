package cocoapods

import (
	"github.com/git-pkgs/manifests/internal/core"
	"regexp"
	"strings"
)

func init() {
	core.Register("cocoapods", core.Manifest, &podfileParser{}, core.ExactMatch("Podfile"))
	core.Register("cocoapods", core.Lockfile, &podfileLockParser{}, core.ExactMatch("Podfile.lock"))
	core.Register("cocoapods", core.Manifest, &podspecParser{}, core.SuffixMatch(".podspec"))
}

// podfileParser parses Podfile manifest files.
type podfileParser struct{}

var (
	// pod "name" or pod "name", "version" or pod "name", "~> 1.0"
	podRegex = regexp.MustCompile(`^\s*pod\s+["']([^"']+)["'](?:\s*,\s*["']([^"']+)["'])?`)
	// target "name" do
	podTargetRegex = regexp.MustCompile(`^\s*target\s+["']([^"']+)["']\s+do`)
)

func (p *podfileParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	inTestTarget := false

	for _, line := range lines {
		// Skip comments
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		// Track test targets
		if match := podTargetRegex.FindStringSubmatch(line); match != nil {
			name := strings.ToLower(match[1])
			inTestTarget = strings.Contains(name, "test")
			continue
		}

		if strings.TrimSpace(line) == "end" {
			inTestTarget = false
			continue
		}

		if match := podRegex.FindStringSubmatch(line); match != nil {
			name := match[1]
			version := ""
			if len(match) > 2 {
				version = match[2]
			}

			scope := core.Runtime
			if inTestTarget {
				scope = core.Test
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

// podfileLockParser parses Podfile.lock files.
type podfileLockParser struct{}

var (
	// PODS section entry: "  - Name (version):" or "  - Name (version)"
	podLockEntryRegex = regexp.MustCompile(`^\s+-\s+([^(]+)\s+\(([^)]+)\)`)
	// SPEC CHECKSUMS entry: "  Name: hash"
	podChecksumRegex = regexp.MustCompile(`^\s+([^:]+):\s+([a-f0-9]+)$`)
)

func (p *podfileLockParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	lines := strings.Split(string(content), "\n")

	var deps []core.Dependency
	checksums := make(map[string]string)
	directDeps := make(map[string]bool)
	section := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect sections
		switch trimmed {
		case "PODS:":
			section = "pods"
			continue
		case "DEPENDENCIES:":
			section = "deps"
			continue
		case "SPEC CHECKSUMS:":
			section = "checksums"
			continue
		case "EXTERNAL SOURCES:", "CHECKOUT OPTIONS:", "COCOAPODS:":
			section = ""
			continue
		}

		if section == "pods" && strings.HasPrefix(line, "  -") && !strings.HasPrefix(line, "    -") {
			// Top-level pod entry
			if match := podLockEntryRegex.FindStringSubmatch(line); match != nil {
				name := strings.TrimSpace(match[1])
				version := match[2]
				deps = append(deps, core.Dependency{
					Name:    name,
					Version: version,
					Scope:   core.Runtime,
					Direct:  false,
				})
			}
		}

		if section == "deps" && strings.HasPrefix(line, "  -") {
			// Direct dependency entry
			name := strings.TrimPrefix(trimmed, "- ")
			// Remove version constraint and other suffixes
			if idx := strings.Index(name, " "); idx > 0 {
				name = name[:idx]
			}
			if idx := strings.Index(name, "("); idx > 0 {
				name = strings.TrimSpace(name[:idx])
			}
			directDeps[name] = true
		}

		if section == "checksums" {
			if match := podChecksumRegex.FindStringSubmatch(line); match != nil {
				checksums[match[1]] = match[2]
			}
		}
	}

	// Update direct status and integrity
	for i := range deps {
		name := deps[i].Name
		// Handle subspecs: "Pod/Subspec" -> check for "Pod"
		baseName := name
		if idx := strings.Index(name, "/"); idx > 0 {
			baseName = name[:idx]
		}
		deps[i].Direct = directDeps[name] || directDeps[baseName]

		if hash, ok := checksums[baseName]; ok {
			deps[i].Integrity = "sha1-" + hash
		}
	}

	return deps, nil
}

// podspecParser parses .podspec files.
type podspecParser struct{}

var (
	// s.dependency "Name" or s.dependency "Name", "version"
	podspecDepRegex = regexp.MustCompile(`\.dependency\s+["']([^"']+)["'](?:\s*,\s*["']([^"']+)["'])?`)
)

func (p *podspecParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)

	for _, match := range podspecDepRegex.FindAllStringSubmatch(text, -1) {
		name := match[1]
		version := ""
		if len(match) > 2 {
			version = match[2]
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
