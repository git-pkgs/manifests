package cargo

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/git-pkgs/manifests/internal/core"
)

const cratesIO = "crates-io"

func init() {
	core.Register("cargo", core.Config, &cargoConfigParser{}, core.CustomMatch(cargoConfigMatch))
}

// cargoConfigMatch matches .cargo/config and .cargo/config.toml at any depth.
func cargoConfigMatch(filename string) bool {
	name := filepath.ToSlash(filename)
	switch path.Base(name) {
	case "config", "config.toml":
	default:
		return false
	}
	dir := path.Dir(name)
	return dir == ".cargo" || strings.HasSuffix(dir, "/.cargo")
}

type cargoSource struct {
	ReplaceWith string `toml:"replace-with"`
	Registry    string `toml:"registry"`
}

type cargoRegistry struct {
	Index string `toml:"index"`
}

// cargoConfigParser reads [source] replacements from Cargo configuration.
type cargoConfigParser struct{}

func (p *cargoConfigParser) Parse(_ string, content []byte) (*core.Result, error) {
	var config struct {
		Source     map[string]cargoSource   `toml:"source"`
		Registries map[string]cargoRegistry `toml:"registries"`
	}
	metadata, err := toml.Decode(string(content), &config)
	if err != nil {
		return nil, err
	}
	var registries []core.Registry
	for _, name := range cargoSourceOrder(metadata) {
		url := replacementRegistry(config.Source, config.Registries, name)
		if url == "" {
			continue
		}
		registry := core.Registry{URL: url, Replaces: true}
		if name != cratesIO {
			registry.Scope = name
		}
		registries = append(registries, registry)
	}
	return &core.Result{Registries: registries}, nil
}

// cargoSourceOrder returns [source.*] table names in file order.
func cargoSourceOrder(metadata toml.MetaData) []string {
	var names []string
	seen := make(map[string]bool)
	for _, key := range metadata.Keys() {
		const sourceTableDepth = 2 // source.<name>
		if len(key) != sourceTableDepth || key[0] != "source" || seen[key[1]] {
			continue
		}
		seen[key[1]] = true
		names = append(names, key[1])
	}
	return names
}

// replacementRegistry follows replace-with from name to the registry URL it
// ends at. A target missing from [source] may name a [registries] entry.
// Directory, local registry and git replacements return "".
func replacementRegistry(sources map[string]cargoSource, registries map[string]cargoRegistry, name string) string {
	next := sources[name].ReplaceWith
	for range len(sources) {
		if next == "" {
			return ""
		}
		target, ok := sources[next]
		if !ok {
			return registries[next].Index
		}
		if target.Registry != "" {
			return target.Registry
		}
		next = target.ReplaceWith
	}
	return ""
}
