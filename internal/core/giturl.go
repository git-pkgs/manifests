package core

import (
	"net/url"
	"strings"
)

// GitCoordinate returns the host/path identity of a Git clone URL, with the
// host lowercased and any .git suffix dropped. It accepts scheme URLs and
// scp-style user@host:path forms and returns "" for relative or local paths,
// whose host is unknown.
func GitCoordinate(rawURL string) string {
	if strings.Contains(rawURL, "://") {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Hostname() == "" {
			return ""
		}

		host := strings.ToLower(parsed.Hostname())
		if parsed.Port() != "" {
			host += ":" + parsed.Port()
		}
		return joinGitCoordinate(host, parsed.Path)
	}

	separator := strings.IndexByte(rawURL, ':')
	if separator <= 0 || strings.ContainsAny(rawURL[:separator], "/\\") {
		return ""
	}
	host := rawURL[:separator]
	if at := strings.LastIndexByte(host, '@'); at >= 0 {
		host = host[at+1:]
	}
	if len(host) == 1 {
		// A single-letter host is a Windows drive prefix, not a remote.
		return ""
	}
	return joinGitCoordinate(strings.ToLower(host), rawURL[separator+1:])
}

func joinGitCoordinate(host, path string) string {
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || path == "" {
		return ""
	}
	return host + "/" + path
}
