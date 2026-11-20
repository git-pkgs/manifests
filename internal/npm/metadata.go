package npm

import (
	"encoding/json"

	"github.com/git-pkgs/manifests/internal/core"
)

// packageJSONMetadata holds the descriptive package.json fields, decoded
// separately so a value of an unexpected type does not fail the parse.
type packageJSONMetadata struct {
	Description json.RawMessage `json:"description"`
	Keywords    json.RawMessage `json:"keywords"`
	Homepage    json.RawMessage `json:"homepage"`
	Repository  json.RawMessage `json:"repository"`
	Bugs        json.RawMessage `json:"bugs"`
	Funding     json.RawMessage `json:"funding"`
}

func (m packageJSONMetadata) metadata() core.Metadata {
	metadata := core.Metadata{
		Description: jsonOptionalString(m.Description),
		Keywords:    jsonStrings(m.Keywords),
		Homepage:    jsonOptionalString(m.Homepage),
	}
	appendNpmLinks(&metadata.Links, "repository", m.Repository)
	appendNpmLinks(&metadata.Links, "bugs", m.Bugs)
	appendNpmLinks(&metadata.Links, "funding", m.Funding)
	return metadata
}

// jsonOptionalString returns the string raw holds, or nil when it is absent
// or not a string.
func jsonOptionalString(raw json.RawMessage) *string {
	var value *string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	return value
}

// jsonStrings returns the strings of a JSON array, non-nil when the array is
// declared, or nil when raw is absent or not an array.
func jsonStrings(raw json.RawMessage) []string {
	var values []any
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return nil
	}
	texts := make([]string, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			texts = append(texts, text)
		}
	}
	return texts
}

// appendNpmLinks appends the URLs of a field that holds a URL string, an
// object with a "url" key, or an array of either, as funding may.
func appendNpmLinks(links *[]core.Link, label string, raw json.RawMessage) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return
	}
	values, ok := value.([]any)
	if !ok {
		values = []any{value}
	}
	for _, value := range values {
		var url string
		switch typed := value.(type) {
		case string:
			url = typed
		case map[string]any:
			url, _ = typed["url"].(string)
		}
		if url != "" {
			*links = append(*links, core.Link{Label: label, URL: url})
		}
	}
}
