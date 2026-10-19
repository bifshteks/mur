package source

import (
	"fmt"
	"slices"

	"gopkg.in/yaml.v3"
)

var (
	workspaceKeys = []string{"summary", "statuses", "repos"}
	statusKeys    = []string{"name", "description", "sync"}
	repoKeys      = []string{"url", "description", "aliases", "langs", "status", "notes"}
)

// UnknownKeys names keys a newer mur (or a person) added that this mur does not
// know. They are kept on rewrite; callers warn about them.
func (w *Workspace) UnknownKeys() []string {
	if w.doc.Kind == 0 {
		return nil
	}

	root := w.doc.Content[0]
	unknown := unknownIn("", root, workspaceKeys)

	for i := 0; i+1 < len(root.Content); i += 2 {
		key, val := root.Content[i].Value, root.Content[i+1]

		switch key {
		case "statuses":
			for j, item := range val.Content {
				unknown = append(unknown, unknownIn(fmt.Sprintf("statuses[%d].", j), item, statusKeys)...)
			}
		case "repos":
			for j, item := range val.Content {
				unknown = append(unknown, unknownIn(fmt.Sprintf("repos[%d].", j), item, repoKeys)...)
			}
		}
	}

	return unknown
}

func unknownIn(prefix string, n *yaml.Node, known []string) []string {
	if n.Kind != yaml.MappingNode {
		return nil
	}

	var out []string

	for i := 0; i < len(n.Content); i += 2 {
		if k := n.Content[i].Value; !slices.Contains(known, k) {
			out = append(out, prefix+k)
		}
	}

	return out
}
