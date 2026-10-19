package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
)

// readWorkspace picks the workspace a read acts on: the flag, then
// $MUR_WORKSPACE, then the active one. A session can pin a workspace without
// touching global state.
func (a *app) readWorkspace(flag string) (*source.Workspace, *state.State, error) {
	name := flag
	if name == "" {
		name = a.env.Getenv("MUR_WORKSPACE")
	}

	if name == "" {
		st, err := a.loadState()
		if err != nil {
			return nil, nil, err
		}

		name = st.Active
	}

	if name == "" {
		return nil, nil, usagef("no workspace: pass --workspace, set MUR_WORKSPACE, or run mur workspace activate")
	}

	return a.findWorkspace(name)
}

type missOutput struct {
	Error      string   `json:"error"      yaml:"error"`
	Query      string   `json:"query"      yaml:"query"`
	Candidates []string `json:"candidates" yaml:"candidates"`
}

// matchOne resolves a query to exactly one repo, or exit 4 with candidates.
func matchOne(w *source.Workspace, query string) (source.Repo, error) {
	hits := w.Match(query)

	switch len(hits) {
	case 1:
		return hits[0], nil
	case 0:
		msg := fmt.Sprintf("no repo named %s in workspace %s", query, w.Qualified())

		return source.Repo{}, &exitError{
			code:   ExitNotFound,
			err:    fmt.Errorf("%s", msg),
			detail: missOutput{Error: msg, Query: query, Candidates: w.Near(query)},
		}
	default:
		msg := fmt.Sprintf("%s matches several repos in workspace %s", query, w.Qualified())
		ids := make([]string, 0, len(hits))

		for _, r := range hits {
			ids = append(ids, r.ID())
		}

		return source.Repo{}, &exitError{
			code:   ExitNotFound,
			err:    fmt.Errorf("%s", msg),
			detail: missOutput{Error: msg, Query: query, Candidates: ids},
		}
	}
}

type needsPathEntry struct {
	ID         string   `json:"id"              yaml:"id"`
	Alias      string   `json:"alias,omitempty" yaml:"alias,omitempty"`
	Candidates []string `json:"candidates"      yaml:"candidates"`
}

type needsPathOutput struct {
	NeedsPath []needsPathEntry `json:"needs_path" yaml:"needs_path"`
	Hint      string           `json:"hint"       yaml:"hint"`
}

func locateHint(w *source.Workspace) string {
	return "mur repo locate <alias|url> <dir> --workspace " + w.Name
}

func (a *app) resolveCommand() *cobra.Command {
	var (
		ws       string
		pathOnly bool
	)

	cmd := &cobra.Command{
		Use:   "resolve <alias|basename|url>...",
		Short: "Show what each named repo is and where it is cloned",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.resolve(ws, args, pathOnly)
		},
	}
	cmd.Flags().StringVar(&ws, "workspace", "", "workspace to resolve in (default $MUR_WORKSPACE, then the active one)")
	cmd.Flags().BoolVar(&pathOnly, "path", false, "print only absolute paths, one per line")

	return cmd
}

func (a *app) resolve(ws string, queries []string, pathOnly bool) error {
	w, st, err := a.readWorkspace(ws)
	if err != nil {
		return err
	}

	views := make([]repoView, 0, len(queries))

	for _, q := range queries {
		r, err := matchOne(w, q)
		if err != nil {
			return err
		}

		v := a.repoView(w, st, &r)
		v.Query = q
		views = append(views, v)
	}

	if !pathOnly {
		return a.print(views)
	}

	var missing []needsPathEntry

	for i := range views {
		if views[i].Path == "" {
			missing = append(missing, needsPathEntry{ID: views[i].ID, Alias: firstOf(views[i].Aliases), Candidates: []string{}})
		}
	}

	if len(missing) > 0 {
		return &exitError{
			code:   ExitNeedsPath,
			err:    fmt.Errorf("no recorded path for %d repo(s)", len(missing)),
			detail: needsPathOutput{NeedsPath: missing, Hint: locateHint(w)},
		}
	}

	for i := range views {
		if _, err := fmt.Fprintln(a.env.Stdout, views[i].Path); err != nil {
			return fmt.Errorf("print path: %w", err)
		}
	}

	return nil
}

func firstOf(s []string) string {
	if len(s) == 0 {
		return ""
	}

	return s[0]
}
