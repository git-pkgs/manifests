package manifests

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func stringPointer(value string) *string { return &value }

func parseMetadataFixture(t *testing.T, path, filename string) *ParseResult {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Parse(filename, content)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// assertMetadata prints both sides as JSON so nil and empty values show apart.
func assertMetadata(t *testing.T, got, want Metadata) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		gotJSON, _ := json.MarshalIndent(got, "", "  ")
		wantJSON, _ := json.MarshalIndent(want, "", "  ")
		t.Errorf("metadata =\n%s\nwant\n%s", gotJSON, wantJSON)
	}
}

func TestPackageJSONMetadata(t *testing.T) {
	result := parseMetadataFixture(t, "testdata/npm/metadata/package.json", "package.json")

	if result.Name != "example" || result.Version != "1.0.0" {
		t.Errorf("identity = %s %s", result.Name, result.Version)
	}
	if !reflect.DeepEqual(result.Licenses, []string{"MIT"}) {
		t.Errorf("licenses = %v", result.Licenses)
	}
	if !reflect.DeepEqual(result.Scripts, map[string][]string{"test": {"node test.js"}}) {
		t.Errorf("scripts = %v", result.Scripts)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "lodash" {
		t.Errorf("dependencies = %+v", result.Dependencies)
	}

	assertMetadata(t, result.Metadata, Metadata{
		Description: stringPointer("A currency conversion library"),
		Keywords:    []string{"currency", "conversion", "汇率"},
		Homepage:    stringPointer("https://example.test"),
		Links: []Link{
			{Label: "repository", URL: "git+https://github.com/example/example.git"},
			{Label: "bugs", URL: "https://github.com/example/example/issues"},
			{Label: "funding", URL: "https://github.com/sponsors/example"},
			{Label: "funding", URL: "https://opencollective.com/example"},
		},
	})
}

func TestPackageJSONMetadataAbsentAndEmpty(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Metadata
	}{
		{
			name:    "absent",
			content: `{"name": "example"}`,
			want:    Metadata{},
		},
		{
			name:    "explicitly empty",
			content: `{"description": "", "keywords": [], "homepage": ""}`,
			want:    Metadata{Description: stringPointer(""), Keywords: []string{}, Homepage: stringPointer("")},
		},
		{
			name:    "null is absent",
			content: `{"description": null, "keywords": null, "homepage": null}`,
			want:    Metadata{},
		},
		{
			name:    "unexpected types are ignored",
			content: `{"description": 1, "keywords": "a b", "homepage": {}, "repository": 2, "bugs": [], "dependencies": {"a": "1.0.0"}}`,
			want:    Metadata{},
		},
		{
			name:    "string repository and bugs",
			content: `{"repository": "github:example/example", "bugs": "https://example.test/issues", "keywords": ["a", 1, "b"]}`,
			want: Metadata{
				Keywords: []string{"a", "b"},
				Links: []Link{
					{Label: "repository", URL: "github:example/example"},
					{Label: "bugs", URL: "https://example.test/issues"},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse("package.json", []byte(test.content))
			if err != nil {
				t.Fatal(err)
			}
			assertMetadata(t, result.Metadata, test.want)
		})
	}
}

func TestNuspecMetadata(t *testing.T) {
	result := parseMetadataFixture(t, "testdata/nuget/metadata.nuspec", "Example.Currency.nuspec")

	if result.Name != "Example.Currency" || result.Version != "2.1.0" {
		t.Errorf("identity = %s %s", result.Name, result.Version)
	}
	if !reflect.DeepEqual(result.Licenses, []string{"MIT"}) {
		t.Errorf("licenses = %v", result.Licenses)
	}
	if len(result.Dependencies) != 1 || result.Dependencies[0].Name != "Newtonsoft.Json" {
		t.Errorf("dependencies = %+v", result.Dependencies)
	}

	assertMetadata(t, result.Metadata, Metadata{
		Summary:     stringPointer("Currency conversion"),
		Description: stringPointer("Converts between currencies.\nSupports ISO 4217 codes."),
		Keywords:    []string{"currency", "conversion", "finance"},
		Homepage:    stringPointer("https://example.test/currency"),
		Readme:      stringPointer("../docs/README.md"),
		Links: []Link{
			{Label: "repository", URL: "https://github.com/example/currency.git"},
			{Label: "iconUrl", URL: "https://example.test/icon.png"},
		},
	})
}

func TestNuspecMetadataAbsentAndEmpty(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Metadata
	}{
		{
			name:    "absent",
			content: `<package><metadata><id>Example</id><version>1.0.0</version></metadata></package>`,
			want:    Metadata{},
		},
		{
			name: "explicitly empty",
			content: `<package><metadata><id>Example</id><version>1.0.0</version>` +
				`<summary></summary><description/><tags>  </tags><projectUrl></projectUrl><readme></readme>` +
				`</metadata></package>`,
			want: Metadata{
				Summary:     stringPointer(""),
				Description: stringPointer(""),
				Keywords:    []string{},
				Homepage:    stringPointer(""),
				Readme:      stringPointer(""),
			},
		},
		{
			name: "legacy license URL",
			content: `<package><metadata><id>Example</id><version>1.0.0</version>` +
				`<licenseUrl>https://example.test/LICENSE</licenseUrl></metadata></package>`,
			want: Metadata{Links: []Link{{Label: "licenseUrl", URL: "https://example.test/LICENSE"}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse("Example.nuspec", []byte(test.content))
			if err != nil {
				t.Fatal(err)
			}
			assertMetadata(t, result.Metadata, test.want)
		})
	}
}

func TestMetadataReadmeIsNotRead(t *testing.T) {
	content := []byte(`<package><metadata><id>Example</id><version>1.0.0</version><readme>missing/README.md</readme></metadata></package>`)
	result, err := Parse("Example.nuspec", content, Options{FSRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if result.Metadata.Readme == nil || *result.Metadata.Readme != "missing/README.md" {
		t.Errorf("readme = %v", result.Metadata.Readme)
	}
	if result.Metadata.Description != nil {
		t.Errorf("description = %q, want absent rather than README content", *result.Metadata.Description)
	}
}
