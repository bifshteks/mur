package source

import (
	"errors"
	"fmt"
	"strings"
)

// Catalog is every attached source, searched together.
type Catalog struct {
	Sources []*Source
}

var ErrNotFound = errors.New("not found")

// AmbiguousError is a bare workspace name defined in more than one source.
type AmbiguousError struct {
	Name       string
	Candidates []string
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("workspace %s is defined in several sources; use one of: %s",
		e.Name, strings.Join(e.Candidates, ", "))
}

// Workspaces loads every workspace in every source.
func (c *Catalog) Workspaces() ([]*Workspace, error) {
	var all []*Workspace

	for _, s := range c.Sources {
		names, err := s.WorkspaceNames()
		if err != nil {
			return nil, err
		}

		for _, n := range names {
			w, err := s.LoadWorkspace(n)
			if err != nil {
				return nil, err
			}

			all = append(all, w)
		}
	}

	return all, nil
}

// Find resolves a bare or source-qualified workspace name.
func (c *Catalog) Find(name string) (*Workspace, error) {
	all, err := c.Workspaces()
	if err != nil {
		return nil, err
	}

	var hits []*Workspace

	for _, w := range all {
		if w.Name == name || w.Qualified() == name {
			hits = append(hits, w)
		}
	}

	switch len(hits) {
	case 0:
		return nil, fmt.Errorf("workspace %s: %w", name, ErrNotFound)
	case 1:
		return hits[0], nil
	default:
		amb := &AmbiguousError{Name: name}
		for _, w := range hits {
			amb.Candidates = append(amb.Candidates, w.Qualified())
		}

		return nil, amb
	}
}

func (c *Catalog) Source(name string) (*Source, bool) {
	for _, s := range c.Sources {
		if s.Name == name {
			return s, true
		}
	}

	return nil, false
}
