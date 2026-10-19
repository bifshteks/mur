package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
)

func (a *app) sourceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "source",
		Short: "Attach, list and update sources: git checkouts that hold workspace definitions",
	}
	cmd.AddCommand(a.sourceInitCommand(), a.sourceAddCommand(), a.sourceListCommand(),
		a.sourcePullCommand(), a.sourceRemoveCommand())

	return cmd
}

func (a *app) sourceInitCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "init <name>",
		Short: "Write a source skeleton into DIR (git init only if DIR is not a repo) and attach it",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.sourceInit(cmd.Context(), args[0], dir)
		},
	}
	cmd.Flags().StringVar(&dir, "path", "", "directory for the source (default $MUR_HOME/sources/<name>)")

	return cmd
}

func (a *app) sourceInit(ctx context.Context, name, dir string) error {
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

	if !a.git.IsRepo(ctx, dir) {
		if _, err := a.git.Run(ctx, dir, "init", "-q"); err != nil {
			return err
		}
	}

	if err := source.Init(dir); err != nil {
		return err
	}

	src := state.Source{Name: name, Path: dir, Remote: a.git.Origin(ctx, dir)}
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
		Use:   cmdList,
		Short: "List attached sources",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			st, err := a.loadState()
			if err != nil {
				return err
			}

			return a.print(nonNil(st.Sources))
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

func (a *app) sourceAddCommand() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "add <name> <git-url|path>",
		Short: "Attach a source: an existing checkout where it is, or a url cloned into DIR",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.sourceAdd(cmd.Context(), args[0], args[1], dir)
		},
	}
	cmd.Flags().StringVar(&dir, "path", "", "where to clone a url (default $MUR_HOME/sources/<name>)")

	return cmd
}

func (a *app) sourceAdd(ctx context.Context, name, from, dir string) error {
	st, err := a.loadState()
	if err != nil {
		return err
	}

	if _, ok := st.Source(name); ok {
		return failf("source %s is already attached", name)
	}

	src := state.Source{Name: name}

	// A checkout already holding mur.yaml is attached in place; anything else
	// (including a bare repo on disk) is a url to clone.
	if local := a.abs(from); fileExists(filepath.Join(local, "mur.yaml")) {
		src.Path = local
		src.Remote = a.git.Origin(ctx, local)
	} else {
		src.Path = a.sourceDir(name, dir)
		src.Remote = from

		if _, err := a.git.Run(ctx, "", "clone", "-q", "--", from, src.Path); err != nil {
			return err
		}
	}

	if _, err := source.Open(name, src.Path); err != nil {
		return err
	}

	st.Sources = append(st.Sources, src)
	if err := a.saveState(st); err != nil {
		return err
	}

	return a.print(src)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)

	return err == nil && !fi.IsDir()
}

type pullResult struct {
	Name   string `json:"name"            yaml:"name"`
	Result string `json:"result"          yaml:"result"`
	Error  string `json:"error,omitempty" yaml:"error,omitempty"`
}

func (a *app) sourcePullCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "pull [name]",
		Short: "Fast-forward sources from their remotes (git pull --ff-only)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.sourcePull(cmd.Context(), firstOf(args))
		},
	}
}

func (a *app) sourcePull(ctx context.Context, only string) error {
	st, err := a.loadState()
	if err != nil {
		return err
	}

	if _, ok := st.Source(only); only != "" && !ok {
		return notFoundf("source %s is not attached", only)
	}

	results := []pullResult{}
	failed := 0

	for _, s := range st.Sources {
		if only != "" && s.Name != only {
			continue
		}

		r := pullResult{Name: s.Name, Result: "pulled"}

		switch s.Remote {
		case "":
			r.Result = "skipped: no remote"
		default:
			if _, err := a.git.Run(ctx, s.Path, "pull", "-q", "--ff-only"); err != nil {
				r.Result, r.Error = "failed", err.Error()
				failed++
			}
		}

		results = append(results, r)
	}

	if err := a.print(results); err != nil {
		return err
	}

	if failed > 0 {
		return &exitError{code: ExitError, err: fmt.Errorf("%d source(s) failed to pull", failed), printed: true}
	}

	return nil
}

func (a *app) sourceRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Detach a source; its files stay where they are",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			st, err := a.loadState()
			if err != nil {
				return err
			}

			i := slices.IndexFunc(st.Sources, func(s state.Source) bool { return s.Name == args[0] })
			if i < 0 {
				return notFoundf("source %s is not attached", args[0])
			}

			removed := st.Sources[i]
			st.Sources = slices.Delete(st.Sources, i, i+1)

			if err := a.saveState(st); err != nil {
				return err
			}

			return a.print(removed)
		},
	}
}
