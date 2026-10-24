package repoid

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var ErrInvalid = errors.New("not a git url")

const schemeFile = "file"

// schemes git clones over; any other scheme:// is a typo or a web page.
//
//nolint:gochecknoglobals // fixed list
var schemes = []string{"ssh", "git+ssh", "ssh+git", "git", "http", "https", schemeFile}

// Validate reports whether url is a remote git can clone and mur can name:
// scheme://host/path, the scp form [user@]host:path, or an absolute path.
// Normalize accepts more (a bare host/path) because queries may be loose; a url
// written into a workspace may not.
func Validate(url string) error {
	if url == "" {
		return fmt.Errorf("%w: empty", ErrInvalid)
	}

	if strings.ContainsFunc(url, unicode.IsSpace) {
		return fmt.Errorf("%w: %q contains whitespace", ErrInvalid, url)
	}

	if scheme, rest, ok := strings.Cut(url, "://"); ok {
		return validateScheme(url, strings.ToLower(scheme), rest)
	}

	if strings.HasPrefix(url, "/") {
		return validatePath(url, url)
	}

	slash := strings.Index(url, "/")
	if colon := strings.Index(url, ":"); colon >= 0 && (slash < 0 || colon < slash) {
		return validateRemote(url, url[:colon], url[colon+1:], false)
	}

	return fmt.Errorf("%w: %q; use git@host:group/repo.git, ssh://, https:// or file://, or an absolute path",
		ErrInvalid, url)
}

func validateScheme(url, scheme, rest string) error {
	known := false

	for _, s := range schemes {
		known = known || s == scheme
	}

	if !known {
		return fmt.Errorf("%w: %q has unknown scheme %s://", ErrInvalid, url, scheme)
	}

	if scheme == schemeFile {
		return validatePath(url, rest)
	}

	hostPart, p, _ := strings.Cut(rest, "/")

	return validateRemote(url, hostPart, p, true)
}

func validateRemote(url, hostPart, p string, portAllowed bool) error {
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}

	host, port, hasPort := strings.Cut(hostPart, ":")
	if host == "" {
		return fmt.Errorf("%w: %q has no host", ErrInvalid, url)
	}

	if hasPort && (!portAllowed || port == "" || strings.ContainsFunc(port, func(r rune) bool { return r < '0' || r > '9' })) {
		return fmt.Errorf("%w: %q has a bad port", ErrInvalid, url)
	}

	if trimPath(p) == "" {
		return fmt.Errorf("%w: %q has no repo path", ErrInvalid, url)
	}

	return nil
}

func validatePath(url, p string) error {
	if !strings.HasPrefix(p, "/") {
		return fmt.Errorf("%w: %q is not an absolute path", ErrInvalid, url)
	}

	if local(p) == "/" {
		return fmt.Errorf("%w: %q has no repo path", ErrInvalid, url)
	}

	return nil
}
