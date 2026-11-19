package pypi

import (
	"bytes"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	core.Register("pypi", core.Config, &pipConfParser{}, core.ExactMatch("pip.conf", "pip.ini"))
}

// pipConfParser reads index settings from pip.conf and pip.ini.
type pipConfParser struct{}

func (p *pipConfParser) Parse(_ string, content []byte) (*core.Result, error) {
	var registries []core.Registry
	seen := make(map[core.Registry]bool)
	add := func(key, value string) {
		replaces := key == "index-url"
		if !replaces && key != "extra-index-url" {
			return
		}
		// extra-index-url may list several URLs separated by white space.
		for _, url := range strings.Fields(value) {
			registry := core.Registry{URL: url, Replaces: replaces}
			if !seen[registry] {
				seen[registry] = true
				registries = append(registries, registry)
			}
		}
	}

	var key, value string
	for line := range bytes.Lines(content) {
		text := strings.TrimRight(string(line), "\r\n")
		trimmed := strings.TrimSpace(text)
		// Like configparser, full-line comments are dropped and blank lines
		// do not end a multi-line value.
		if trimmed == "" || trimmed[0] == '#' || trimmed[0] == ';' {
			continue
		}
		// configparser continues a value on indented lines.
		if key != "" && (text[0] == ' ' || text[0] == '\t') {
			value += " " + trimmed
			continue
		}
		if key != "" {
			add(key, value)
			key = ""
		}
		if trimmed[0] == '[' {
			continue
		}
		separator := strings.IndexAny(trimmed, "=:")
		if separator <= 0 {
			continue
		}
		// pip accepts index_url and INDEX-URL as index-url.
		key = strings.ReplaceAll(strings.ToLower(strings.TrimSpace(trimmed[:separator])), "_", "-")
		value = strings.TrimSpace(trimmed[separator+1:])
	}
	if key != "" {
		add(key, value)
	}
	return &core.Result{Registries: registries}, nil
}
