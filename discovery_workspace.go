package manifests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"go.yaml.in/yaml/v3"
)

func (d *manifestDiscovery) discoverCargoWorkspace() error {
	const parentPath = "Cargo.toml"
	content, exists, err := d.readOptional(parentPath)
	if err != nil {
		return fmt.Errorf("reading Cargo workspace configuration: %w", err)
	}
	if !exists {
		return nil
	}

	var config struct {
		Workspace struct {
			Members []string `toml:"members"`
			Exclude []string `toml:"exclude"`
		} `toml:"workspace"`
	}
	if _, err := toml.Decode(string(content), &config); err != nil {
		return fmt.Errorf("parsing Cargo workspace configuration: %w", err)
	}
	if err := d.addWorkspaceManifests(
		config.Workspace.Members,
		config.Workspace.Exclude,
		"Cargo.toml",
		parentPath,
	); err != nil {
		return fmt.Errorf("discovering Cargo workspace members: %w", err)
	}
	return nil
}

func (d *manifestDiscovery) discoverGoWorkspace() error {
	const parentPath = "go.work"
	content, exists, err := d.readOptional(parentPath)
	if err != nil {
		return fmt.Errorf("reading Go workspace configuration: %w", err)
	}
	if !exists {
		return nil
	}

	members := goWorkspaceMembers(content)
	if err := d.addWorkspaceManifests(members, nil, "go.mod", parentPath); err != nil {
		return fmt.Errorf("discovering Go workspace members: %w", err)
	}
	return nil
}

func goWorkspaceMembers(content []byte) []string {
	var members []string
	inUseBlock := false
	for line := range strings.SplitSeq(string(content), "\n") {
		line = stripGoWorkComment(line)
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		if inUseBlock {
			var closed bool
			members, closed = appendGoWorkUseFields(members, fields)
			inUseBlock = !closed
			continue
		}

		if fields[0] != "use" {
			continue
		}
		if len(fields) >= 2 && fields[1] == "(" {
			var closed bool
			members, closed = appendGoWorkUseFields(members, fields[2:])
			inUseBlock = !closed
			continue
		}
		members, _ = appendGoWorkUseFields(members, fields[1:])
	}
	return members
}

func appendGoWorkUseFields(members, fields []string) ([]string, bool) {
	for _, field := range fields {
		switch field {
		case "(":
			continue
		case ")":
			return members, true
		default:
			members = append(members, strings.Trim(field, `"'`))
		}
	}
	return members, false
}

func stripGoWorkComment(line string) string {
	if index := strings.Index(line, "//"); index >= 0 {
		line = line[:index]
	}
	return strings.TrimSpace(line)
}

func (d *manifestDiscovery) discoverNPMWorkspace() error {
	const parentPath = "package.json"
	content, exists, err := d.readOptional(parentPath)
	if err != nil {
		return fmt.Errorf("reading npm workspace configuration: %w", err)
	}
	if !exists {
		return nil
	}

	patterns, err := npmWorkspacePatterns(content)
	if err != nil {
		return fmt.Errorf("parsing npm workspace configuration: %w", err)
	}
	if err := d.addWorkspaceManifests(patterns, nil, "package.json", parentPath); err != nil {
		return fmt.Errorf("discovering npm workspace members: %w", err)
	}
	return nil
}

func npmWorkspacePatterns(content []byte) ([]string, error) {
	var config struct {
		Workspaces json.RawMessage `json:"workspaces"`
	}
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}
	if len(config.Workspaces) == 0 || bytes.Equal(bytes.TrimSpace(config.Workspaces), []byte("null")) {
		return nil, nil
	}

	var patterns []string
	if err := json.Unmarshal(config.Workspaces, &patterns); err == nil {
		return patterns, nil
	}
	var grouped struct {
		Packages []string `json:"packages"`
	}
	if err := json.Unmarshal(config.Workspaces, &grouped); err != nil {
		return nil, err
	}
	return grouped.Packages, nil
}

func (d *manifestDiscovery) discoverPnpmWorkspace() error {
	const parentPath = "pnpm-workspace.yaml"
	content, exists, err := d.readOptional(parentPath)
	if err != nil {
		return fmt.Errorf("reading pnpm workspace configuration: %w", err)
	}
	if !exists {
		return nil
	}

	var config struct {
		Packages []string `yaml:"packages"`
	}
	if err := yaml.Unmarshal(content, &config); err != nil {
		return fmt.Errorf("parsing pnpm workspace configuration: %w", err)
	}

	includes := make([]string, 0, len(config.Packages))
	excludes := make([]string, 0, len(config.Packages))
	for _, pattern := range config.Packages {
		if len(pattern) > 0 && pattern[0] == '!' {
			excludes = append(excludes, pattern[1:])
			continue
		}
		includes = append(includes, pattern)
	}
	if err := d.addWorkspaceManifests(includes, excludes, "package.json", parentPath); err != nil {
		return fmt.Errorf("discovering pnpm workspace members: %w", err)
	}
	return nil
}
