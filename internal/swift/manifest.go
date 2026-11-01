package swift

import (
	"path"
	"strconv"
	"strings"

	"github.com/git-pkgs/manifests/internal/core"
)

type packageSwiftParser struct{}

func (p *packageSwiftParser) Parse(_ string, content []byte) (*core.Result, error) {
	tokens := tokenize(string(content))
	result := &core.Result{}
	seen := make(map[string]bool)
	bindings := make(map[string]string)
	var packageArgs []argument
	for i, token := range tokens {
		if token.literal || (token.text != "package" && token.text != "Package") {
			continue
		}
		args, ok := callArguments(tokens[i+1:])
		if !ok {
			continue
		}
		if i == 0 || !tokens[i-1].is(".") {
			if token.text == "Package" {
				result.Name = argumentString(args, "name")
				packageArgs = args
			}
			continue
		}
		dep := packageDependency(args)
		if dep.Name == "" {
			// Path packages have no entry, but targets can still reference them.
			if local := argumentString(args, "path"); local != "" {
				bindPackage(bindings, args, path.Base(local), local)
			}
			continue
		}
		if seen[dep.Name] {
			continue
		}
		seen[dep.Name] = true
		result.Dependencies = append(result.Dependencies, dep)
		identity := dep.Name
		if dep.Source.Kind == core.SourceGit {
			identity = path.Base(identity)
		}
		bindPackage(bindings, args, identity, dep.Name)
	}
	applyTargetScopes(result.Dependencies, bindings, packageArgs)
	return result, nil
}

func packageDependency(args []argument) core.Dependency {
	dep := core.Dependency{Scope: core.Runtime, Direct: true}
	if repository := argumentString(args, "url"); repository != "" {
		dep.Name = core.GitCoordinate(repository)
		dep.Source = core.Source{Kind: core.SourceGit, Value: repository}
	} else if identity := argumentString(args, "id"); identity != "" {
		dep.Name = strings.ToLower(identity)
		dep.Source = core.Source{Kind: core.SourceRegistry, Value: identity}
	}
	for _, arg := range args {
		kind, version := requirement(arg)
		switch kind {
		case "branch":
			dep.Source.Branch = version
		case "revision":
			dep.Source.Ref = version
		}
		if version != "" {
			dep.Version = version
		}
	}
	return dep
}

func requirement(arg argument) (string, string) {
	kind := arg.label
	value := literalValue(arg.value)
	if kind == "" {
		if bounds := rangeRequirement(arg.value); bounds != "" {
			return "range", bounds
		}
		if len(arg.value) < 2 || !arg.value[0].is(".") || arg.value[1].literal {
			return "", ""
		}
		kind = arg.value[1].text
		args, ok := callArguments(arg.value[2:])
		if !ok || len(args) != 1 {
			return "", ""
		}
		value = literalValue(args[0].value)
	}
	if value == "" {
		return "", ""
	}
	switch kind {
	case "exact", "branch", "revision":
		return kind, value
	case "from", "upToNextMajor":
		return kind, nextVersionRange(value, false)
	case "upToNextMinor":
		return kind, nextVersionRange(value, true)
	default:
		return "", ""
	}
}

func nextVersionRange(version string, minor bool) string {
	const versionParts = 3
	parts := strings.SplitN(version, ".", versionParts)
	if len(parts) != versionParts {
		return ""
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil || major < 0 {
		return ""
	}
	if !minor {
		return ">=" + version + " <" + strconv.Itoa(major+1) + ".0.0"
	}
	minorVersion, err := strconv.Atoi(parts[1])
	if err != nil || minorVersion < 0 {
		return ""
	}
	return ">=" + version + " <" + parts[0] + "." + strconv.Itoa(minorVersion+1) + ".0"
}

func rangeRequirement(tokens []token) string {
	const rangeTokens = 5
	if len(tokens) != rangeTokens || !tokens[0].literal || !tokens[4].literal ||
		!tokens[1].is(".") || !tokens[2].is(".") {
		return ""
	}
	operator := "<"
	switch {
	case tokens[3].is("."):
		operator = "<="
	case !tokens[3].is("<"):
		return ""
	}
	return ">=" + tokens[0].text + " " + operator + tokens[4].text
}

type argument struct {
	label string
	value []token
}

func callArguments(tokens []token) ([]argument, bool) {
	args, end := delimitedArguments(tokens, "(", ")")
	return args, end > 0
}

func delimitedArguments(tokens []token, open, close string) ([]argument, int) {
	if len(tokens) == 0 || !tokens[0].is(open) {
		return nil, 0
	}
	var args []argument
	start, depth := 1, 0
	for i := 1; i < len(tokens); i++ {
		t := tokens[i]
		if t.literal {
			continue
		}
		if depth == 0 && (t.is(",") || t.is(close)) {
			if i > start {
				args = append(args, makeArgument(tokens[start:i]))
			}
			if t.is(close) {
				return args, i + 1
			}
			start = i + 1
		}
		switch t.text {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
		}
	}
	return nil, 0
}

func makeArgument(tokens []token) argument {
	if len(tokens) > 1 && !tokens[0].literal && tokens[1].is(":") {
		return argument{label: tokens[0].text, value: tokens[2:]}
	}
	return argument{value: tokens}
}

func argumentString(args []argument, label string) string {
	return literalValue(argumentValue(args, label))
}

func argumentValue(args []argument, label string) []token {
	for _, arg := range args {
		if arg.label == label {
			return arg.value
		}
	}
	return nil
}

func literalValue(tokens []token) string {
	if len(tokens) == 1 && tokens[0].literal {
		return tokens[0].text
	}
	return ""
}
