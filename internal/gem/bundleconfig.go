package gem

import (
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/git-pkgs/manifests/internal/core"
)

const bundleMirrorPrefix = "BUNDLE_MIRROR__"

func init() {
	core.Register("gem", core.Config, &bundleConfigParser{}, core.CustomMatch(bundleConfigMatch))
}

// bundleConfigMatch matches .bundle/config at any depth.
func bundleConfigMatch(filename string) bool {
	name := filepath.ToSlash(filename)
	dir := path.Dir(name)
	return path.Base(name) == "config" && (dir == ".bundle" || strings.HasSuffix(dir, "/.bundle"))
}

// bundleConfigParser reads mirror settings from Bundler's .bundle/config.
// Other keys, including host credentials, are ignored.
type bundleConfigParser struct{}

func (p *bundleConfigParser) Parse(_ string, content []byte) (*core.Result, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, err
	}
	root := &document
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return &core.Result{}, nil
	}
	var registries []core.Registry
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i].Value, root.Content[i+1]
		source, ok := strings.CutPrefix(key, bundleMirrorPrefix)
		if !ok || source == "" || strings.HasSuffix(source, "__FALLBACK_TIMEOUT") || value.Kind != yaml.ScalarNode {
			continue
		}
		registry := core.Registry{URL: value.Value, Replaces: true}
		if source != "ALL" {
			registry.Scope = bundleKeySource(source)
		}
		registries = append(registries, registry)
	}
	return &core.Result{Registries: registries}, nil
}

// bundleKeySource reverses Bundler's key encoding, which upcases the source
// and writes "-" as "___" and "." as "__".
func bundleKeySource(encoded string) string {
	decoded := strings.ReplaceAll(strings.ToLower(encoded), "___", "-")
	return strings.ReplaceAll(decoded, "__", ".")
}
