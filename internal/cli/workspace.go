package cli

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
)

func (a *app) workspaceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workspace",
		Short: "List, create, activate and show workspaces: named sets of repos with a README",
	}
	cmd.AddCommand(a.workspaceListCommand(), a.workspaceCreateCommand())

	return cmd
}

type workspaceEntry struct {
	Name    string `json:"name"    yaml:"name"`
	Source  string `json:"source"  yaml:"source"`
	Summary string `json:"summary" yaml:"summary"`
}

func (a *app) workspaceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workspaces across all sources",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			cat, _, err := a.catalog()
			if err != nil {
				return fail(err)
			}

			all, err := cat.Workspaces()
			if err != nil {
				return fail(err)
			}

			out := []workspaceEntry{}
			for _, w := range all {
				out = append(out, workspaceEntry{Name: w.Name, Source: w.Source, Summary: w.Summary})
			}

			return fail(a.print(out))
		},
	}
}

func (a *app) workspaceCreateCommand() *cobra.Command {
	var srcName, summary string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a workspace in a source (--source required when several are attached)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return fail(a.workspaceCreate(args[0], srcName, summary))
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
