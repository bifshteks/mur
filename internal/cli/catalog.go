package cli

import (
	"errors"
	"fmt"

	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
)

// catalog opens every attached source.
func (a *app) catalog() (*source.Catalog, *state.State, error) {
	st, err := a.loadState()
	if err != nil {
		return nil, nil, err
	}

	cat := &source.Catalog{}

	for _, s := range st.Sources {
		src, err := source.Open(s.Name, s.Path)
		if err != nil {
			return nil, nil, err
		}

		cat.Sources = append(cat.Sources, src)
	}

	return cat, st, nil
}

type ambiguousOutput struct {
	Error      string   `json:"error"      yaml:"error"`
	Candidates []string `json:"candidates" yaml:"candidates"`
}

// findWorkspace looks a workspace up by bare or qualified name, mapping a miss
// or a clash between sources to exit 4.
func (a *app) findWorkspace(name string) (*source.Workspace, *state.State, error) {
	cat, st, err := a.catalog()
	if err != nil {
		return nil, nil, err
	}

	w, err := cat.Find(name)

	var amb *source.AmbiguousError

	switch {
	case errors.As(err, &amb):
		return nil, nil, &exitError{
			code:   ExitNotFound,
			err:    amb,
			detail: ambiguousOutput{Error: amb.Error(), Candidates: amb.Candidates},
		}
	case errors.Is(err, source.ErrNotFound):
		return nil, nil, &exitError{code: ExitNotFound, err: err}
	case err != nil:
		return nil, nil, err
	}

	return w, st, nil
}

// warnUnknown tells the user which keys a rewrite kept without understanding.
func (a *app) warnUnknown(w *source.Workspace) {
	for _, k := range w.UnknownKeys() {
		_, _ = fmt.Fprintf(a.env.Stderr, "warning: %s/workspace.yaml: unknown key %q kept\n", w.Qualified(), k)
	}
}
