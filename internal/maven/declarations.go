package maven

import (
	"net/url"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
	"github.com/git-pkgs/pom"
)

const defaultMavenPluginGroup = "org.apache.maven.plugins"

func parsePOMDeclarations(project *pom.POM) []core.Declaration {
	var declarations []core.Declaration
	if project.Parent != nil {
		parent := project.Parent
		appendMavenDeclaration(
			&declarations,
			"parent",
			parent.GroupID,
			parent.ArtifactID,
			parent.Version,
			core.Build,
			"",
		)
	}
	collectPOMDependencies(&declarations, "dependencies", project.Dependencies)
	collectPOMDependencies(&declarations, "dependencyManagement/dependencies", project.DependencyManagement.Dependencies)
	collectMavenPlugins(&declarations, "build/plugins", project.Build.Plugins)
	collectMavenPlugins(&declarations, "build/pluginManagement/plugins", project.Build.PluginManagement.Plugins)
	collectMavenExtensions(&declarations, "build/extensions", project.Build.Extensions)

	for _, profile := range project.Profiles {
		profileID := strings.TrimSpace(profile.ID)
		if profileID == "" {
			profileID = "default"
		}
		prefix := "profiles/" + url.PathEscape(profileID)
		collectPOMDependencies(&declarations, prefix+"/dependencies", profile.Dependencies)
		collectPOMDependencies(&declarations, prefix+"/dependencyManagement/dependencies", profile.DependencyManagement.Dependencies)
		collectMavenPlugins(&declarations, prefix+"/build/plugins", profile.Build.Plugins)
		collectMavenPlugins(&declarations, prefix+"/build/pluginManagement/plugins", profile.Build.PluginManagement.Plugins)
		collectMavenExtensions(&declarations, prefix+"/build/extensions", profile.Build.Extensions)
	}

	return declarations
}

func collectPOMDependencies(declarations *[]core.Declaration, location string, dependencies []pom.Dep) {
	for _, dependency := range dependencies {
		optional := strings.EqualFold(strings.TrimSpace(dependency.Optional), "true")
		appendMavenDeclaration(
			declarations,
			location,
			dependency.GroupID,
			dependency.ArtifactID,
			dependency.Version,
			mapScope(dependency.Scope, optional),
			"",
			dependency.Type,
			dependency.Classifier,
		)
	}
}

func collectMavenPlugins(declarations *[]core.Declaration, location string, plugins []pom.Plugin) {
	for _, plugin := range plugins {
		pluginName := appendMavenDeclaration(
			declarations,
			location,
			plugin.GroupID,
			plugin.ArtifactID,
			plugin.Version,
			core.Build,
			defaultMavenPluginGroup,
		)
		if pluginName == "" {
			continue
		}
		dependencyLocation := location + "/" + url.PathEscape(pluginName) + "/dependencies"
		for _, dependency := range plugin.Dependencies {
			appendMavenDeclaration(
				declarations,
				dependencyLocation,
				dependency.GroupID,
				dependency.ArtifactID,
				dependency.Version,
				core.Build,
				"",
			)
		}
	}
}

func collectMavenExtensions(declarations *[]core.Declaration, location string, extensions []pom.Extension) {
	for _, extension := range extensions {
		appendMavenDeclaration(
			declarations,
			location,
			extension.GroupID,
			extension.ArtifactID,
			extension.Version,
			core.Build,
			"",
		)
	}
}

func appendMavenDeclaration(
	declarations *[]core.Declaration,
	location string,
	groupID string,
	artifactID string,
	version string,
	scope core.Scope,
	defaultGroup string,
	qualifiers ...string,
) string {
	groupID = strings.TrimSpace(groupID)
	artifactID = strings.TrimSpace(artifactID)
	if artifactID == "" {
		return ""
	}
	if groupID == "" {
		groupID = defaultGroup
	}
	if groupID == "" {
		return ""
	}

	name := groupID + ":" + artifactID
	key := name
	for _, qualifier := range qualifiers {
		if qualifier = strings.TrimSpace(qualifier); qualifier != "" {
			key += ":" + qualifier
		}
	}
	*declarations = append(*declarations, core.Declaration{
		Name:     name,
		Version:  strings.TrimSpace(version),
		Scope:    scope,
		Location: location + "/" + url.PathEscape(key),
	})
	return name
}
