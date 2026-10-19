package repoid

import (
	"path"
	"strings"
)

// Normalize turns any spelling of a git remote into the repo's identity:
// host/path for network remotes, the absolute path for local ones. Ssh, https
// and scp forms of one repo yield the same id; the url as written is kept
// elsewhere, because that is what gets cloned.
func Normalize(url string) string {
	s := strings.TrimSpace(url)
	if s == "" {
		return ""
	}

	if scheme, rest, ok := strings.Cut(s, "://"); ok {
		if strings.EqualFold(scheme, "file") {
			return local(rest)
		}

		hostPart, p, _ := strings.Cut(rest, "/")

		return remote(hostPart, p)
	}

	if strings.HasPrefix(s, "/") {
		return local(s)
	}

	slash := strings.Index(s, "/")
	if colon := strings.Index(s, ":"); colon >= 0 && (slash < 0 || colon < slash) {
		return remote(s[:colon], s[colon+1:])
	}

	hostPart, p, _ := strings.Cut(s, "/")

	return remote(hostPart, p)
}

func remote(hostPart, p string) string {
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}

	host, _, _ := strings.Cut(hostPart, ":")

	return strings.ToLower(host) + "/" + trimPath(p)
}

func local(p string) string {
	return "/" + trimPath(path.Clean(p))
}

func trimPath(p string) string {
	p = strings.Trim(p, "/")
	p = strings.TrimSuffix(p, ".git")

	return strings.TrimRight(p, "/")
}
