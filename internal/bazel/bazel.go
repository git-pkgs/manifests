package bazel

import (
	"fmt"
	"regexp"

	"github.com/bazelbuild/buildtools/build"
	"github.com/git-pkgs/manifests/internal/core"
)

func init() {
	// MODULE.bazel - manifest
	core.Register("bazel", core.Manifest, &bazelModuleManifestParser{}, core.ExactMatch("MODULE.bazel"))

}

type bazelModuleManifestParser struct{}

func (p *bazelModuleManifestParser) Parse(filename string, content []byte) (*core.Result, error) {
	var deps []core.Dependency
	var selfName, selfVersion string

	moduleManifestTree, err := build.ParseModule(filename, content)
	if err != nil {
		return nil, &core.ParseError{Filename: filename, Err: err}
	}

	for _, expression := range moduleManifestTree.Stmt {
		// Dependency statements - bazel_dep() - have CallExpr type
		callExperssion, ok := expression.(*build.CallExpr)
		if !ok {
			continue
		}

		ident, ok := callExperssion.X.(*build.Ident)
		if !ok {
			continue
		}

		// module(name = "...", version = "...")
		if ident.Name == "module" {
			selfName, selfVersion = parseBazelModule(*callExperssion)
			continue
		}

		// Check if statement is a dependency declaration
		if ident.Name != "bazel_dep" {
			continue
		}

		parsedDep, err := parseBazelDep(*callExperssion)
		if err != nil {
			return nil, &core.ParseError{Filename: filename, Err: err}
		}

		scope := core.Build
		if parsedDep.DevDependency {
			scope = core.Development
		}
		deps = append(deps, core.Dependency{
			Name:    parsedDep.Name,
			Version: parsedDep.Version,
			Scope:   scope,
			Direct:  true,
		})

	}
	return &core.Result{Name: selfName, Version: selfVersion, Dependencies: deps}, nil
}

func parseBazelModule(call build.CallExpr) (name, version string) {
	for _, arg := range call.List {
		assign, ok := arg.(*build.AssignExpr)
		if !ok || assign == nil {
			continue
		}
		key, ok := assign.LHS.(*build.Ident)
		if !ok {
			continue
		}
		val, ok := assign.RHS.(*build.StringExpr)
		if !ok {
			continue
		}
		switch key.Name {
		case "name":
			name = val.Value
		case "version":
			version = val.Value
		}
	}
	return name, version
}

type bazelDep struct {
	Name          string
	Version       string
	DevDependency bool
}

type bazelDepParsingError struct {
	Message string
	Line    int
}

var moduleNameRegex = regexp.MustCompile(`^[a-z]([a-z0-9._-]*[a-z0-9])?$`)

func parseBazelDep(callExperssion build.CallExpr) (*bazelDep, error) {
	dep := &bazelDep{
		DevDependency: false, // default
	}

	// Check each argument of a dependency statement
	//   e.g. bazel_dep(name = "google_benchmark", version = "1.9.4", dev_dependency = True,)
	for _, arg := range callExperssion.List {
		assign, ok := arg.(*build.AssignExpr)
		if !ok || assign == nil {
			continue
		}

		keyIdent, ok := assign.LHS.(*build.Ident)
		if !ok {
			continue
		}
		switch keyIdent.Name {
		case "name":
			nameExpr, ok := assign.RHS.(*build.StringExpr)
			if !ok {
				return nil, &bazelDepParsingError{
					Line:    callExperssion.ListStart.Line,
					Message: "bazel_dep 'name' attribute is not a string"}
			}

			if !moduleNameRegex.MatchString(nameExpr.Value) {
				return nil, &bazelDepParsingError{
					Line:    callExperssion.ListStart.Line,
					Message: fmt.Sprintf("bazel_dep 'name' %q has invalid format", nameExpr.Value)}
			}
			dep.Name = nameExpr.Value

		case "version":
			if versionExpr, ok := assign.RHS.(*build.StringExpr); ok {
				dep.Version = versionExpr.Value
			}
		case "dev_dependency":
			if devDependencyExpr, ok := assign.RHS.(*build.Ident); ok {
				dep.DevDependency = devDependencyExpr.Name == "True"
			}
		}
	}
	return dep, nil
}

func (e *bazelDepParsingError) Error() string {
	return fmt.Sprintf(
		"%d:%s",
		e.Line,
		e.Message,
	)
}
