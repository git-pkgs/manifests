package core

import "strings"

// ForEachLine iterates over lines in content without allocating a slice.
func ForEachLine(content string, fn func(line string) bool) {
	for len(content) > 0 {
		idx := strings.IndexByte(content, '\n')
		var line string
		if idx < 0 {
			line = content
			content = ""
		} else {
			line = content[:idx]
			content = content[idx+1:]
		}
		if !fn(line) {
			return
		}
	}
}

// EstimateDeps estimates the number of dependencies based on file size.
func EstimateDeps(size int) int {
	estimate := size / 50
	if estimate < 4 {
		return 4
	}
	if estimate > 1000 {
		return 1000
	}
	return estimate
}

// ExtractQuotedValue extracts value from lines like: key = "value"
func ExtractQuotedValue(line, prefix string) (string, bool) {
	if !strings.HasPrefix(line, prefix) {
		return "", false
	}
	rest := line[len(prefix):]
	if len(rest) < 2 || rest[0] != '"' {
		return "", false
	}
	end := strings.IndexByte(rest[1:], '"')
	if end < 0 {
		return "", false
	}
	return rest[1 : end+1], true
}

// ParseDockerImage parses a Docker image reference like "nginx:1.19" or "nginx@sha256:abc".
// When both tag and digest are present (nginx:1.19@sha256:abc), the digest is used as version
// and the tag is stripped from the name.
func ParseDockerImage(image string) (name, version string) {
	// Handle digest format: image@sha256:abc or image:tag@sha256:abc
	if idx := strings.Index(image, "@"); idx > 0 {
		namePart := image[:idx]
		version = image[idx+1:]

		// Strip tag from name if present (e.g., node:6@sha256:... -> node)
		if tagIdx := strings.LastIndex(namePart, ":"); tagIdx > 0 {
			// Make sure we're not stripping a port or registry (e.g., registry.io:5000/image)
			afterColon := namePart[tagIdx+1:]
			if !strings.Contains(afterColon, "/") {
				namePart = namePart[:tagIdx]
			}
		}
		return namePart, version
	}

	// Handle tag format: image:tag
	if idx := strings.LastIndex(image, ":"); idx > 0 {
		// Make sure we're not stripping a port (e.g., registry.io:5000/image)
		afterColon := image[idx+1:]
		if !strings.Contains(afterColon, "/") {
			return image[:idx], image[idx+1:]
		}
	}

	// No tag or digest, default to latest
	return image, "latest"
}
