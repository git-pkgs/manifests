package core

import "regexp"

var shellInstallPattern = regexp.MustCompile(`(?m)^install=(?:"([^"\\]*)"|'([^']*)'|([^\s#;'"\\]+))[\t ]*(?:#.*)?\r?$`)

// ShellInstall extracts a top-level install assignment without shell expansion.
func ShellInstall(content string) string {
	match := shellInstallPattern.FindStringSubmatch(content)
	if match == nil {
		return ""
	}
	for _, value := range match[1:] {
		if value != "" {
			return value
		}
	}
	return ""
}
