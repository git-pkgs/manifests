package swift

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
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
	// Package(name: "...")
	swiftSelfNameRegex = regexp.MustCompile(`\bPackage\s*\(\s*name:\s*"([^"]+)"`)
)

func (p *packageSwiftParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	text := string(content)
	seen := make(map[string]bool)

	var selfName string
	if m := swiftSelfNameRegex.FindStringSubmatch(text); m != nil {
		selfName = m[1]
	}

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
			const versionGroup = 2
			if len(match) > versionGroup {
				version = match[versionGroup]
			}

			name := swiftSourceCoordinate(url)
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

	return &core.Result{Name: selfName, Dependencies: deps}, nil
}

func swiftSourceCoordinate(rawURL string) string {
	if strings.Contains(rawURL, "://") {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Hostname() == "" {
			return ""
		}

		host := strings.ToLower(parsed.Hostname())
		if parsed.Port() != "" {
			host += ":" + parsed.Port()
		}
		return joinSwiftSourceCoordinate(host, parsed.Path)
	}

	separator := strings.IndexByte(rawURL, ':')
	if separator <= 0 {
		return ""
	}
	host := rawURL[:separator]
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		host = host[at+1:]
	}
	return joinSwiftSourceCoordinate(strings.ToLower(host), rawURL[separator+1:])
}

func joinSwiftSourceCoordinate(host, path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || path == "" {
		return ""
	}
	return host + "/" + path
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
	Kind     string `json:"kind"`
	Location string `json:"location"`
	State    struct {
		Version  string `json:"version"`
		Revision string `json:"revision"`
	} `json:"state"`
}

func (p *packageResolvedParser) Parse(filename string, content []byte) (*core.Result, error) {
	// Try to detect version
	var versionCheck struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(content, &versionCheck); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	const resolvedV2 = 2
	var deps []core.Dependency
	var err error
	if versionCheck.Version >= resolvedV2 {
		deps, err = parsePackageResolvedV2(filename, content)
	} else {
		deps, err = parsePackageResolvedV1(filename, content)
	}
	return &core.Result{Dependencies: deps}, err
}

func parsePackageResolvedV1(filename string, content []byte) ([]core.Dependency, error) {
	var resolved packageResolvedV1
	if err := json.Unmarshal(content, &resolved); err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	var deps []core.Dependency
	for _, pin := range resolved.Object.Pins {
		name := swiftSourceCoordinate(pin.RepositoryURL)
		if name == "" {
			name = pin.Package
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
		if pin.Kind == "remoteSourceControl" {
			if coordinate := swiftSourceCoordinate(pin.Location); coordinate != "" {
				name = coordinate
			}
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
