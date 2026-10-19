package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
)

func (a *app) workspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "List, create, activate and show workspaces: named sets of repos with a README",
	}
	cmd.AddCommand(a.workspaceListCommand(), a.workspaceCreateCommand(), a.workspaceActivateCommand(), a.workspaceShowCommand())

	return cmd
}

type workspaceEntry struct {
	Name    string     `json:"name"             yaml:"name"`
	Source  string     `json:"source"           yaml:"source"`
	Summary string     `json:"summary"          yaml:"summary"`
	Active  bool       `json:"active,omitempty" yaml:"active,omitempty"`
	Repos   []repoView `json:"repos,omitempty"  yaml:"repos,omitempty"`
}

func (a *app) workspaceListCommand() *cobra.Command {
	var withRepos bool

	cmd := &cobra.Command{
		Use:   cmdList,
		Short: "List workspaces across all sources",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cat, st, err := a.catalog()
			if err != nil {
				return err
			}

			all, err := cat.Workspaces()
			if err != nil {
				return err
			}

			active := ""
			if w, err := cat.Find(st.Active); err == nil {
				active = w.Qualified()
			}

			out := []workspaceEntry{}

			for _, w := range all {
				e := workspaceEntry{Name: w.Name, Source: w.Source, Summary: w.Summary, Active: w.Qualified() == active}
				if withRepos {
					e.Repos = a.repoViews(cmd.Context(), w, st)
				}

				out = append(out, e)
			}

			return a.print(out)
		},
	}
	cmd.Flags().BoolVar(&withRepos, "repos", false, "include each workspace's repos")

	return cmd
}

func (a *app) workspaceCreateCommand() *cobra.Command {
	var srcName, summary string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a workspace in a source (--source required when several are attached)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.workspaceCreate(args[0], srcName, summary)
		},
	}
	cmd.Flags().StringVar(&srcName, "source", "", "source to create the workspace in")
	cmd.Flags().StringVar(&summary, "summary", "", "one-line summary shown by workspace list")

	return cmd
}

func (a *app) workspaceCreate(name, srcName, summary string) error {
	cat, _, err := a.catalog()
	if err != nil {
		return err
	}

	var src *source.Source

	switch {
	case srcName != "":
		s, ok := cat.Source(srcName)
		if !ok {
			return notFoundf("source %s is not attached", srcName)
		}

		src = s
	case len(cat.Sources) == 1:
		src = cat.Sources[0]
	case len(cat.Sources) == 0:
		return failf("no source attached; run mur source init or mur source add first")
	default:
		return usagef("several sources are attached; pass --source")
	}

	w, err := src.CreateWorkspace(name, summary)
	if errors.Is(err, source.ErrExists) {
		return failf("workspace %s/%s already exists", src.Name, name)
	}

	if err != nil {
		return err
	}

	return a.print(workspaceEntry{Name: w.Name, Source: w.Source, Summary: w.Summary})
}

type activeOutput struct {
	Active string `json:"active" yaml:"active"`
}

func (a *app) workspaceActivateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "activate <name>",
		Short: "Make a workspace the default for reads",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			w, st, err := a.findWorkspace(args[0])
			if err != nil {
				return err
			}

			st.Active = args[0]
			if err := a.saveState(st); err != nil {
				return err
			}

			return a.print(activeOutput{Active: w.Qualified()})
		},
	}
}

type workspaceShowOutput struct {
	Name     string          `json:"name"     yaml:"name"`
	Source   string          `json:"source"   yaml:"source"`
	Dir      string          `json:"dir"      yaml:"dir"`
	Summary  string          `json:"summary"  yaml:"summary"`
	Readme   string          `json:"readme"   yaml:"readme"`
	Statuses []source.Status `json:"statuses" yaml:"statuses"`
	Repos    []repoView      `json:"repos"    yaml:"repos"`
	Files    []string        `json:"files"    yaml:"files"`
}

func (a *app) workspaceShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Everything to read before working across a workspace: summary, README, repos with paths, extra files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.workspaceShow(cmd.Context(), firstOf(args))
		},
	}
}

func (a *app) workspaceShow(ctx context.Context, name string) error {
	w, st, err := a.readWorkspace(name)
	if err != nil {
		return err
	}

	readme, err := w.Readme()
	if err != nil {
		return err
	}

	files, err := w.ExtraFiles()
	if err != nil {
		return err
	}

	out := workspaceShowOutput{
		Name:     w.Name,
		Source:   w.Source,
		Dir:      w.Dir,
		Summary:  w.Summary,
		Readme:   readme,
		Statuses: nonNil(w.Statuses),
		Repos:    []repoView{},
		Files:    files,
	}

	for i := range w.Repos {
		v := a.repoView(ctx, w, st, &w.Repos[i])
		v.Workspace = ""
		out.Repos = append(out.Repos, v)
	}

	return a.print(out)
}
