package cli

import (
	"os"
	"path/filepath"

	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
)

// repoView is how every command shows a repo: its definition joined with where
// it is on this laptop.
type repoView struct {
	Query       string   `json:"query,omitempty"     yaml:"query,omitempty"`
	Workspace   string   `json:"workspace,omitempty" yaml:"workspace,omitempty"`
	ID          string   `json:"id"                  yaml:"id"`
	URL         string   `json:"url"                 yaml:"url"`
	Path        string   `json:"path,omitempty"      yaml:"path,omitempty"`
	Cloned      bool     `json:"cloned"              yaml:"cloned"`
	Description string   `json:"description"         yaml:"description"`
	Aliases     []string `json:"aliases,omitempty"   yaml:"aliases,omitempty"`
	Langs       []string `json:"langs,omitempty"     yaml:"langs,omitempty"`
	Status      string   `json:"status,omitempty"    yaml:"status,omitempty"`
	Notes       string   `json:"notes,omitempty"     yaml:"notes,omitempty"`
}

func (a *app) repoView(w *source.Workspace, st *state.State, r *source.Repo) repoView {
	path := st.Paths[r.ID()]

	return repoView{
		Workspace:   w.Qualified(),
		ID:          r.ID(),
		URL:         r.URL,
		Path:        path,
		Cloned:      path != "" && isClone(path),
		Description: r.Description,
		Aliases:     r.Aliases,
		Langs:       r.Langs,
		Status:      r.Status,
		Notes:       r.Notes,
	}
}

// isClone is what sync counts as cloned: an empty or missing directory is
// still a place to clone into.
func isClone(p string) bool {
	_, err := os.Stat(filepath.Join(p, ".git"))

	return err == nil
}

func isDir(p string) bool {
	fi, err := os.Stat(p)

	return err == nil && fi.IsDir()
}

// repoViews shows every repo of a workspace, without repeating the workspace.
func (a *app) repoViews(w *source.Workspace, st *state.State) []repoView {
	views := make([]repoView, 0, len(w.Repos))

	for i := range w.Repos {
		v := a.repoView(w, st, &w.Repos[i])
		v.Workspace = ""
		views = append(views, v)
	}

	return views
}
