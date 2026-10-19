package cli

import (
	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
)

func (a *app) repoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Add, list, remove and locate the repos of a workspace",
	}
	cmd.AddCommand(a.repoAddCommand(), a.repoLocateCommand())

	return cmd
}

func (a *app) repoAddCommand() *cobra.Command {
	var (
		ws string
		r  source.Repo
	)

	cmd := &cobra.Command{
		Use:   "add <url>",
		Short: "Add a repo to a workspace (writes workspace.yaml; --workspace required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if ws == "" {
				return usagef("repo add writes a source; pass --workspace")
			}

			r.URL = args[0]

			return fail(a.repoAdd(ws, r))
		},
	}
	cmd.Flags().StringVar(&ws, "workspace", "", "workspace to add the repo to")
	cmd.Flags().StringVar(&r.Description, "description", "", "what the repo is for (required)")
	cmd.Flags().StringArrayVar(&r.Aliases, "alias", nil, "workspace-scoped name; repeatable")
	cmd.Flags().StringArrayVar(&r.Langs, "lang", nil, "language or stack, free text; repeatable")
	cmd.Flags().StringVar(&r.Status, "status", "", "one of the statuses the workspace declares")
	cmd.Flags().StringVar(&r.Notes, "notes", "", "operational rules for working in the repo")
	_ = cmd.MarkFlagRequired("description")

	return cmd
}

func (a *app) repoAdd(ws string, r source.Repo) error {
	w, st, err := a.findWorkspace(ws)
	if err != nil {
		return err
	}

	a.warnUnknown(w)

	if err := w.AddRepo(r); err != nil {
		return err
	}

	return a.print(a.repoView(w, st, r))
}

type locatedOutput struct {
	ID     string `json:"id"     yaml:"id"`
	Path   string `json:"path"   yaml:"path"`
	Cloned bool   `json:"cloned" yaml:"cloned"`
}

func (a *app) repoLocateCommand() *cobra.Command {
	var ws string

	cmd := &cobra.Command{
		Use:   "locate <alias|url> <dir>",
		Short: "Record where a repo is (or should be) cloned on this laptop; dir may not exist yet",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return fail(a.repoLocate(ws, args[0], args[1]))
		},
	}
	cmd.Flags().StringVar(&ws, "workspace", "", "workspace the repo is in (default $MUR_WORKSPACE, then the active one)")

	return cmd
}

func (a *app) repoLocate(ws, query, dir string) error {
	w, st, err := a.readWorkspace(ws)
	if err != nil {
		return err
	}

	r, err := matchOne(w, query)
	if err != nil {
		return err
	}

	p := a.abs(dir)
	st.SetPath(r.ID(), p)

	if err := a.saveState(st); err != nil {
		return err
	}

	return a.print(locatedOutput{ID: r.ID(), Path: p, Cloned: isDir(p)})
}
