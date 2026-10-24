package source

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

// AddStatus declares a status in workspace.yaml, rewriting only the statuses
// node. Sync is written only when false, the one value that changes anything.
func (w *Workspace) AddStatus(s Status) error {
	if s.Name == "" || s.Description == "" {
		return fmt.Errorf("%w: name and description are required", ErrInvalid)
	}

	if strings.ContainsFunc(s.Name, unicode.IsSpace) {
		return fmt.Errorf("%w: status name %q contains whitespace", ErrInvalid, s.Name)
	}

	if w.declares(s.Name) {
		return fmt.Errorf("%w: status %s is already declared in workspace %s", ErrInvalid, s.Name, w.Name)
	}

	if s.Sync != nil && *s.Sync {
		s.Sync = nil
	}

	statuses := w.seqNode(keyStatuses)
	statuses.Content = append(statuses.Content, statusNode(s))

	w.moveBefore(keyStatuses, keyRepos)

	if err := writeYAML(w.file(), &w.doc); err != nil {
		return err
	}

	w.Statuses = append(w.Statuses, s)

	return nil
}

func statusNode(s Status) *yaml.Node {
	n := mapping(scalar("name"), scalar(s.Name), scalar(keyDescription), scalar(s.Description))
	if s.Sync != nil {
		n.Content = append(n.Content, scalar("sync"), &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "false"})
	}

	return n
}

// moveBefore puts key right before other when key sits after it, so a status
// list mur creates reads above the repos that use it. Hand-written order is
// otherwise left alone.
func (w *Workspace) moveBefore(key, other string) {
	root := w.root()
	ki, oi := -1, -1

	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case key:
			ki = i
		case other:
			oi = i
		}
	}

	if ki < 0 || oi < 0 || ki < oi {
		return
	}

	pair := []*yaml.Node{root.Content[ki], root.Content[ki+1]}
	root.Content = slices.Delete(root.Content, ki, ki+2)
	root.Content = slices.Insert(root.Content, oi, pair...)
}
