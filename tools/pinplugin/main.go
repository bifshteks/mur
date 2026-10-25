// Command pinplugin pins the mur plugin in .claude-plugin/marketplace.json to a
// release tag, so the skill a user installs is the one that tag holds (ADR 0005).
// Run it from the repo root:
//
//	go run ./tools/pinplugin <tag> <sha>
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	marketplacePath = ".claude-plugin/marketplace.json"
	pluginName      = "mur"
	repoURL         = "https://github.com/bifshteks/mur.git"
)

// ErrNotNewer means the marketplace already pins a newer release than the tag:
// a hotfix on an older line, or an old release's job re-run after a newer one.
var ErrNotNewer = errors.New("marketplace already pins a newer release")

type marketplace struct {
	Name     string          `json:"name"`
	Owner    json.RawMessage `json:"owner"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	Plugins  []plugin        `json:"plugins"`
}

type plugin struct {
	Name        string          `json:"name"`
	Version     string          `json:"version,omitempty"`
	Source      json.RawMessage `json:"source"`
	Description string          `json:"description,omitempty"`
}

type gitSource struct {
	Source string `json:"source"`
	URL    string `json:"url"`
	Ref    string `json:"ref"`
	SHA    string `json:"sha"`
}

// Pin returns marketplace with the mur plugin's source set to tag at sha and its
// version set to the tag without the leading "v". Every other field is kept.
func Pin(data []byte, tag, sha string) ([]byte, error) {
	// Stable releases only; the workflow skips pin-plugin for prerelease tags.
	if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(tag) {
		return nil, fmt.Errorf("tag %q is not vX.Y.Z", tag)
	}

	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(sha) {
		return nil, fmt.Errorf("sha %q is not a full commit hash", sha)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	// A field this tool does not know would be dropped on write; refuse instead.
	dec.DisallowUnknownFields()

	var m marketplace
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("read marketplace: %w", err)
	}

	source, err := json.Marshal(gitSource{Source: "url", URL: repoURL, Ref: tag, SHA: sha})
	if err != nil {
		return nil, err
	}

	pinned := false

	for i := range m.Plugins {
		if m.Plugins[i].Name == pluginName {
			if older(tag[1:], m.Plugins[i].Version) {
				return nil, fmt.Errorf("%w: %s, not %s", ErrNotNewer, m.Plugins[i].Version, tag[1:])
			}

			m.Plugins[i].Version = tag[1:]
			m.Plugins[i].Source = source
			pinned = true
		}
	}

	if !pinned {
		return nil, fmt.Errorf("no plugin %q in marketplace", pluginName)
	}

	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(out, '\n'), nil
}

// older reports whether X.Y.Z version a is lower than b. A b that is not
// X.Y.Z, such as none at all before the first release, never blocks a pin.
func older(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	if len(bs) != len(as) {
		return false
	}

	for i := range as {
		x, _ := strconv.Atoi(as[i])
		y, _ := strconv.Atoi(bs[i])

		if x != y {
			return x < y
		}
	}

	return false
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: pinplugin <tag> <sha>")
		os.Exit(2)
	}

	tag, sha := os.Args[1], os.Args[2]

	data, err := os.ReadFile(marketplacePath)
	if err == nil {
		data, err = Pin(data, tag, sha)
	}

	if errors.Is(err, ErrNotNewer) {
		fmt.Println("pinplugin: left as is:", err)

		return
	}

	if err == nil {
		err = os.WriteFile(marketplacePath, data, 0o600) //nolint:gosec // fixed path; tag and sha are validated by Pin
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "pinplugin:", err)
		os.Exit(1)
	}
}
