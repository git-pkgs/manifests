package asdf

import (
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("asdf", core.Manifest, &toolVersionsParser{}, core.ExactMatch(".tool-versions"))
}

// toolVersionsParser parses .tool-versions files.
type toolVersionsParser struct{}

func (p *toolVersionsParser) Parse(filename string, content []byte) ([]core.Dependency, error) {
	var deps []core.Dependency
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		version := fields[1]

		deps = append(deps, core.Dependency{
			Name:    name,
			Version: version,
			Scope:   core.Runtime,
			Direct:  true,
		})
	}

	return deps, nil
}
