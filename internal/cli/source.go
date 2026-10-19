package cli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
)

func (a *app) sourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Attach, list and update sources: git checkouts that hold workspace definitions",
	}
	cmd.AddCommand(a.sourceInitCommand(), a.sourceListCommand())

	return cmd
}

func (a *app) sourceInitCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Write a source skeleton into DIR (git init only if DIR is not a repo) and attach it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fail(a.sourceInit(cmd, args[0], dir))
		},
	}
	cmd.Flags().StringVar(&dir, "path", "", "directory for the source (default $MUR_HOME/sources/<name>)")

	return cmd
}

func (a *app) sourceInit(cmd *cobra.Command, name, dir string) error {
	st, err := a.loadState()
	if err != nil {
		return err
	}

	if _, ok := st.Source(name); ok {
		return failf("source %s is already attached", name)
	}

	dir = a.sourceDir(name, dir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return failf("create source dir: %w", err)
	}

	if !a.git.IsRepo(cmd.Context(), dir) {
		if _, err := a.git.Run(cmd.Context(), dir, "init", "-q"); err != nil {
			return err
		}
	}

	if err := source.Init(dir); err != nil {
		return err
	}

	src := state.Source{Name: name, Path: dir, Remote: a.git.Origin(cmd.Context(), dir)}
	st.Sources = append(st.Sources, src)

	if err := a.saveState(st); err != nil {
		return err
	}

	return a.print(src)
}

func (a *app) sourceDir(name, dir string) string {
	if dir == "" {
		return filepath.Join(a.murHome(), "sources", name)
	}

	return a.abs(dir)
}

func (a *app) sourceListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List attached sources",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := a.loadState()
			if err != nil {
				return fail(err)
			}

			return fail(a.print(nonNil(st.Sources)))
		},
	}
}

// nonNil keeps an empty list printing as [] rather than null.
func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}

	return s
}
