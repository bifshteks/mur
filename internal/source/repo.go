package source

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bifshteks/mur/internal/repoid"
)

func (r Repo) ID() string {
	return repoid.Normalize(r.URL)
}

var ErrInvalid = errors.New("invalid repo")

// AddRepo appends a repo to workspace.yaml, rewriting only the repos node.
func (w *Workspace) AddRepo(r Repo) error {
	if err := w.validateNew(r); err != nil {
		return err
	}

	repos := w.reposNode()
	repos.Content = append(repos.Content, repoNode(r))

	if err := writeYAML(w.file(), &w.doc); err != nil {
		return err
	}

	w.Repos = append(w.Repos, r)

	return nil
}

func (w *Workspace) validateNew(r Repo) error {
	if r.URL == "" || r.Description == "" {
		return fmt.Errorf("%w: url and description are required", ErrInvalid)
	}

	for _, have := range w.Repos {
		if have.ID() == r.ID() {
			return fmt.Errorf("%w: %s is already in workspace %s", ErrInvalid, r.ID(), w.Name)
		}

		for _, a := range r.Aliases {
			if slices.Contains(have.Aliases, a) {
				return fmt.Errorf("%w: alias %s already names %s in workspace %s", ErrInvalid, a, have.ID(), w.Name)
			}
		}
	}

	if r.Status != "" && !w.declares(r.Status) {
		return fmt.Errorf("%w: status %s is not declared in workspace %s", ErrInvalid, r.Status, w.Name)
	}

	return nil
}

func (w *Workspace) declares(status string) bool {
	for _, s := range w.Statuses {
		if s.Name == status {
			return true
		}
	}

	return false
}

// root is the top-level mapping, created when the file was empty.
func (w *Workspace) root() *yaml.Node {
	if w.doc.Kind == 0 {
		w.doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{mapping()}}
	}

	return w.doc.Content[0]
}

func (w *Workspace) reposNode() *yaml.Node {
	root := w.root()

	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "repos" {
			seq := root.Content[i+1]
			if seq.Kind != yaml.SequenceNode {
				*seq = yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			}

			seq.Style &^= yaml.FlowStyle

			return seq
		}
	}

	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	root.Content = append(root.Content, scalar("repos"), seq)

	return seq
}

func repoNode(r Repo) *yaml.Node {
	n := mapping(scalar("url"), scalar(r.URL), scalar("description"), scalar(r.Description))

	if len(r.Aliases) > 0 {
		n.Content = append(n.Content, scalar("aliases"), sequence(r.Aliases))
	}

	if len(r.Langs) > 0 {
		n.Content = append(n.Content, scalar("langs"), sequence(r.Langs))
	}

	if r.Status != "" {
		n.Content = append(n.Content, scalar("status"), scalar(r.Status))
	}

	if r.Notes != "" {
		n.Content = append(n.Content, scalar("notes"), scalar(r.Notes))
	}

	return n
}

// Match finds the repos a query names, trying in turn an alias, a basename and
// any spelling of the url. The first level with a hit wins, so an alias can
// claim a name another repo's basename also has.
func (w *Workspace) Match(query string) []Repo {
	levels := []func(Repo) bool{
		func(r Repo) bool { return slices.Contains(r.Aliases, query) },
		func(r Repo) bool { return path.Base(r.ID()) == query },
		func(r Repo) bool { return r.ID() == repoid.Normalize(query) },
	}

	for _, matches := range levels {
		var hits []Repo

		for _, r := range w.Repos {
			if matches(r) {
				hits = append(hits, r)
			}
		}

		if len(hits) > 0 {
			return hits
		}
	}

	return nil
}

// Near lists repos whose alias or basename contains the query: what the caller
// may have meant.
func (w *Workspace) Near(query string) []string {
	near := []string{}

	for _, r := range w.Repos {
		names := append([]string{path.Base(r.ID())}, r.Aliases...)
		if slices.ContainsFunc(names, func(n string) bool { return strings.Contains(n, query) }) {
			near = append(near, r.ID())
		}
	}

	return near
}

// Alias is the repo's first alias, or empty.
func (r Repo) Alias() string {
	if len(r.Aliases) == 0 {
		return ""
	}

	return r.Aliases[0]
}
