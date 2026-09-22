package core

// StringScripts accepts string commands and ordered lists of string commands.
func StringScripts(values map[string]any) map[string][]string {
	var scripts map[string][]string
	for name, value := range values {
		switch commands := value.(type) {
		case string:
			AddScript(&scripts, name, commands)
		case []any:
			for _, command := range commands {
				if text, ok := command.(string); ok {
					AddScript(&scripts, name, text)
				}
			}
		}
	}
	return scripts
}

// AddScript preserves command order and omits empty declarations.
func AddScript(scripts *map[string][]string, name string, commands ...string) {
	for _, command := range commands {
		if command == "" {
			continue
		}
		if *scripts == nil {
			*scripts = make(map[string][]string)
		}
		(*scripts)[name] = append((*scripts)[name], command)
	}
}
