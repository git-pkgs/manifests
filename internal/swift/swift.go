package swift

import (
	"github.com/git-pkgs/manifests/internal/core"
	"encoding/json"
	"regexp"
	"strings"
)

func init() {
	core.Register("swift", core.Manifest, &packageSwiftParser{}, core.ExactMatch("Package.swift"))
	core.Register("swift", core.Lockfile, &packageResolvedParser{}, core.ExactMatch("Package.resolved"))
}

// packageSwiftParser parses Package.swift files.
type packageSwiftParser struct{}

var (
	// .Package(url: "https://...", majorVersion: 0, minor: 12)
	swiftPackageV3Regex = regexp.MustCompile(`\.Package\s*\(\s*url:\s*"([^"]+)"`)
	// .package(url: "https://...", from: "1.0.0")
	swiftPackageV4FromRegex = regexp.MustCompile(`\.package\s*\(\s*url:\s*"([^"]+)",\s*from:\s*"([^"]+)"`)
	// .package(url: "https://...", .upToNextMajor(from: "1.0.0"))
	swiftPackageV4UpToRegex = regexp.MustCompile(`\.package\s*\(\s*url:\s*"([^"]+)",\s*\.upToNextMajor\s*\(\s*from:\s*"([^"]+)"`)
	// .package(url: "https://...", "1.0.0"..<"2.0.0")
	swiftPackageV4RangeRegex = regexp.MustCompile(`\.package\s*\(\s*url:\s*"([^"]+)",\s*"([^"]+)"`)
	// .package(name: "...", url: "https://...", from: "1.0.0")
	swiftPackageNamedRegex = regexp.MustCompile(`\.package\s*\(\s*(?:name:\s*"[^"]+",\s*)?url:\s*"([^"]+)"`)
)

func (p *packageSwiftParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	// Try different regex patterns
	for _, regex := range []*regexp.Regexp{
		swiftPackageV4FromRegex,
		swiftPackageV4UpToRegex,
		swiftPackageV4RangeRegex,
		swiftPackageV3Regex,
		swiftPackageNamedRegex,
	} {
		for _, match := range regex.FindAllStringSubmatch(text, -1) {
			url := match[1]
			version := ""
			if len(match) > 2 {
				version = match[2]
			}

			name := extractSwiftPackageName(url)
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true

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

// extractSwiftPackageName extracts the package name from a git URL.
func extractSwiftPackageName(url string) string {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Get last path component
	if idx := strings.LastIndex(url, "/"); idx >= 0 {
		return url[idx+1:]
	}
	return url
}

// packageResolvedParser parses Package.resolved files.
type packageResolvedParser struct{}

type packageResolvedV1 struct {
	Object struct {
		Pins []packageResolvedPinV1 `json:"pins"`
	} `json:"object"`
	Version int `json:"version"`
}

type packageResolvedPinV1 struct {
	Package       string `json:"package"`
	RepositoryURL string `json:"repositoryURL"`
	State         struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"state"`
}

type packageResolvedV2 struct {
	Pins    []packageResolvedPinV2 `json:"pins"`
	Version int                    `json:"version"`
}

type packageResolvedPinV2 struct {
	Identity string `json:"identity"`
	Location string `json:"location"`
	State    struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"state"`
}

func (p *packageResolvedParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	// Try to detect version
	var versionCheck struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(content, &versionCheck); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	if versionCheck.Version >= 2 {
		return parsePackageResolvedV2(filename, content)
	}
	return parsePackageResolvedV1(filename, content)
}

func parsePackageResolvedV1(filename string, content []byte) ([]core.Dependency, error) {
	var resolved packageResolvedV1
	if err := json.Unmarshal(content, &resolved); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, pin := range resolved.Object.Pins {
		name := pin.Package
		if name == "" {
			name = extractSwiftPackageName(pin.RepositoryURL)
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: pin.State.Version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}

func parsePackageResolvedV2(filename string, content []byte) ([]core.Dependency, error) {
	var resolved packageResolvedV2
	if err := json.Unmarshal(content, &resolved); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, pin := range resolved.Pins {
		name := pin.Identity
		if name == "" {
			name = extractSwiftPackageName(pin.Location)
		}

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: pin.State.Version,
			Scope:   core.Runtime,
			Direct:  false,
		})
	}

	return deps, nil
}
