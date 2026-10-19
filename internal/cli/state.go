package cli

import (
	"context"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
)

func (a *app) stateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Share this laptop's state through a source, for a new laptop to pull",
	}
	cmd.AddCommand(a.statePushCommand(), a.statePullCommand())

	return cmd
}

type stateFlags struct {
	source string
	user   string
}

func (f *stateFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.source, "source", "", "source to write to or read from (required)")
	cmd.Flags().StringVar(&f.user, "user", "", "key under users/ (default: git config user.email in the source)")
	_ = cmd.MarkFlagRequired("source")
}

// sharedStatePath is users/<email>/state.yaml in the source.
func (a *app) sharedStatePath(ctx context.Context, st *state.State, f stateFlags) (string, error) {
	src, ok := st.Source(f.source)
	if !ok {
		return "", notFoundf("source %s is not attached", f.source)
	}

	user := f.user
	if user == "" {
		email, err := a.git.Run(ctx, src.Path, "config", "user.email")
		if err != nil || email == "" {
			return "", usagef("no git user.email in source %s; pass --user", f.source)
		}

		user = email
	}

	return filepath.Join(src.Path, "users", user, "state.yaml"), nil
}

type pushOutput struct {
	Path string `json:"path" yaml:"path"`
}

func (a *app) statePushCommand() *cobra.Command {
	var f stateFlags

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Write this laptop's state into users/<email>/state.yaml of a source (no commit)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			st, err := a.loadState()
			if err != nil {
				return err
			}

			p, err := a.sharedStatePath(cmd.Context(), st, f)
			if err != nil {
				return err
			}

			if err := st.WithTilde(a.env.Getenv("HOME")).Save(p); err != nil {
				return err
			}

			return a.print(pushOutput{Path: p})
		},
	}
	f.bind(cmd)

	return cmd
}

type pullOutput struct {
	Sources   []string `json:"sources"              yaml:"sources"`
	Paths     []string `json:"paths"                yaml:"paths"`
	Active    string   `json:"active,omitempty"     yaml:"active,omitempty"`
	ScanRoots []string `json:"scan_roots,omitempty" yaml:"scan_roots,omitempty"`
}

func (a *app) statePullCommand() *cobra.Command {
	var f stateFlags

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Merge users/<email>/state.yaml from a source: local entries win, missing sources are cloned",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.statePull(cmd.Context(), f)
		},
	}
	f.bind(cmd)

	return cmd
}

func (a *app) statePull(ctx context.Context, f stateFlags) error {
	st, err := a.loadState()
	if err != nil {
		return err
	}

	p, err := a.sharedStatePath(ctx, st, f)
	if err != nil {
		return err
	}

	if !fileExists(p) {
		return notFoundf("no shared state at %s", p)
	}

	shared, err := state.Load(p)
	if err != nil {
		return err
	}

	merged := st.Merge(shared.Expanded(a.env.Getenv("HOME")))
	out := pullOutput{Sources: []string{}, Paths: nonNil(merged.Paths), Active: merged.Active, ScanRoots: merged.ScanRoots}

	for _, src := range merged.Sources {
		if err := a.attachShared(ctx, src); err != nil {
			return err
		}

		st.Sources = append(st.Sources, src)
		out.Sources = append(out.Sources, src.Name)
	}

	if err := a.saveState(st); err != nil {
		return err
	}

	return a.print(out)
}

// attachShared makes a source from another laptop usable here: in place when
// it is already checked out at its path, cloned there otherwise.
func (a *app) attachShared(ctx context.Context, src state.Source) error {
	if !fileExists(filepath.Join(src.Path, "mur.yaml")) {
		if src.Remote == "" {
			return failf("source %s has no remote and is not at %s", src.Name, src.Path)
		}

		if _, err := a.git.Run(ctx, "", "clone", "-q", "--", src.Remote, src.Path); err != nil {
			return failf("source %s: clone %s: %w", src.Name, src.Remote, err)
		}
	}

	_, err := source.Open(src.Name, src.Path)

	return err
}
