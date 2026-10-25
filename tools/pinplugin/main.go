// Command pinplugin pins the mur plugin in .claude-plugin/marketplace.json to a
// release tag, so the skill a user installs is the one that tag holds (ADR 0005).
// Run it from the repo root:
//
//	go run ./tools/pinplugin <tag> <sha>
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
)

const (
	marketplacePath = ".claude-plugin/marketplace.json"
	pluginName      = "mur"
	repoURL         = "https://github.com/bifshteks/mur.git"
)

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
	// The same tags the release workflow runs on: v*.*.*, no prereleases.
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

	if err == nil {
		err = os.WriteFile(marketplacePath, data, 0o600) //nolint:gosec // fixed path; tag and sha are validated by Pin
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "pinplugin:", err)
		os.Exit(1)
	}
}
