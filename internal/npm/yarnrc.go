package npm

import (
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("npm", core.Config, &yarnRCParser{}, core.ExactMatch(".yarnrc.yml"))
}

// yarnRCParser reads registry settings from a Yarn Berry .yarnrc.yml.
type yarnRCParser struct{}

func (p *yarnRCParser) Parse(_ string, content []byte) (*core.Result, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, err
	}
	var registries []core.Registry
	root := yamlMapping(&document)
	if server := yamlScalar(yamlValue(root, "npmRegistryServer")); server != "" {
		registries = append(registries, core.Registry{URL: server, Replaces: true})
	}
	if scopes := yamlValue(root, "npmScopes"); scopes != nil && scopes.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(scopes.Content); i += 2 {
			name := strings.TrimPrefix(scopes.Content[i].Value, "@")
			server := yamlScalar(yamlValue(scopes.Content[i+1], "npmRegistryServer"))
			if name == "" || server == "" {
				continue
			}
			registries = append(registries, core.Registry{URL: server, Scope: "@" + name, Replaces: true})
		}
	}
	return &core.Result{Registries: registries}, nil
}

// yamlMapping returns the mapping at the root of a document or nil.
func yamlMapping(node *yaml.Node) *yaml.Node {
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	return node
}

// yamlValue returns the value for key in mapping or nil.
func yamlValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func yamlScalar(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return strings.TrimSpace(node.Value)
}
