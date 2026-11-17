package manifests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestNPMWorkspaceLinks(t *testing.T) {
	for _, version := range []int{2, 3} {
		content := []byte(fmt.Sprintf(`{
  "lockfileVersion": %d,
  "packages": {
    "": {
      "dependencies": {"workspace-lib": "file:packages/workspace-lib", "registry-lib": "^2.0.0"},
      "devDependencies": {"@scope/local": "file:packages/local"}
    },
    "node_modules/workspace-lib": {"resolved": "packages/workspace-lib", "link": true},
    "node_modules/@scope/local": {"resolved": "packages/local", "link": true, "dev": true},
    "node_modules/external": {"resolved": "../external", "link": true, "optional": true},
    "node_modules/registry-lib": {"version": "2.1.0", "resolved": "https://registry.npmjs.org/registry-lib/-/registry-lib-2.1.0.tgz", "integrity": "sha512-synthetic"},
    "packages/workspace-lib": {"version": "1.0.0"},
    "packages/local": {"version": "3.0.0"},
    "packages/some_node_modules/not-installed": {"version": "6.0.0"},
    "../external": {"version": "4.0.0"},
    "packages/workspace-lib/node_modules/nested": {"version": "5.0.0", "resolved": "https://registry.npmjs.org/nested/-/nested-5.0.0.tgz"}
  }
}`, version))
		var compact, indented bytes.Buffer
		if err := json.Compact(&compact, content); err != nil {
			t.Fatal(err)
		}
		if err := json.Indent(&indented, compact.Bytes(), "", "  "); err != nil {
			t.Fatal(err)
		}
		for _, filename := range []string{"package-lock.json", "npm-shrinkwrap.json"} {
			for format, input := range map[string][]byte{"compact": compact.Bytes(), "indented": indented.Bytes()} {
				t.Run(fmt.Sprintf("v%d/%s/%s", version, filename, format), func(t *testing.T) {
					result, err := Parse(filename, input)
					if err != nil {
						t.Fatal(err)
					}
					got := make(map[string]Dependency)
					for _, dep := range result.Dependencies {
						if _, duplicate := got[dep.Name]; duplicate {
							t.Errorf("duplicate dependency %q", dep.Name)
						}
						got[dep.Name] = dep
					}
					want := map[string]Dependency{
						"workspace-lib": {Name: "workspace-lib", Scope: Runtime, Direct: true, Ecosystem: "npm", PURL: "pkg:npm/workspace-lib", Source: Source{Kind: SourcePath, Value: "packages/workspace-lib"}},
						"@scope/local":  {Name: "@scope/local", Scope: Development, Direct: true, Ecosystem: "npm", PURL: "pkg:npm/%40scope/local", Source: Source{Kind: SourcePath, Value: "packages/local"}},
						"external":      {Name: "external", Scope: Optional, Ecosystem: "npm", PURL: "pkg:npm/external", Source: Source{Kind: SourcePath, Value: "../external"}},
						"registry-lib":  {Name: "registry-lib", Version: "2.1.0", Scope: Runtime, Direct: true, Ecosystem: "npm", PURL: "pkg:npm/registry-lib@2.1.0", RegistryURL: "https://registry.npmjs.org/registry-lib/-/registry-lib-2.1.0.tgz", Integrity: "sha512-synthetic"},
						"nested":        {Name: "nested", Version: "5.0.0", Scope: Runtime, Ecosystem: "npm", PURL: "pkg:npm/nested@5.0.0", RegistryURL: "https://registry.npmjs.org/nested/-/nested-5.0.0.tgz"},
					}
					if !reflect.DeepEqual(got, want) {
						t.Errorf("dependencies = %#v\nwant %#v", got, want)
					}
				})
			}
		}
	}
}
