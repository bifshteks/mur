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

func (f *stateFlags) bind(cmd *cobra.Command, sourceUsage string) {
	cmd.Flags().StringVar(&f.source, "source", "", sourceUsage)
	cmd.Flags().StringVar(&f.user, "user", "", "key under users/ (default: git config user.email in the source)")
}

// sharedStatePath is users/<email>/state.yaml in the source.
func (a *app) sharedStatePath(ctx context.Context, st *state.State, name, user string) (string, error) {
	src, ok := st.Source(name)
	if !ok {
		return "", notFoundf("source %s is not attached", name)
	}

	if user == "" {
		email, err := a.git.Run(ctx, src.Path, "config", "user.email")
		if err != nil || email == "" {
			return "", usagef("no git user.email in source %s; pass --user", name)
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
		Short: "Write the part of this laptop's state that belongs to a source into its users/<email>/state.yaml (no commit)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cat, st, err := a.catalog()
			if err != nil {
				return err
			}

			src, ok := cat.Source(f.source)
			if !ok {
				return notFoundf("source %s is not attached", f.source)
			}

			p, err := a.sharedStatePath(cmd.Context(), st, f.source, f.user)
			if err != nil {
				return err
			}

			repoIDs, workspaces, err := sourceScope(src)
			if err != nil {
				return err
			}

			if err := st.ScopedTo(repoIDs, workspaces).WithTilde(a.env.Getenv("HOME")).Save(p); err != nil {
				return err
			}

			return a.print(pushOutput{Path: p})
		},
	}
	f.bind(cmd, "source to write to (required)")
	_ = cmd.MarkFlagRequired("source")

	return cmd
}

// sourceScope is what a source can know about this laptop: the ids of the
// repos its workspaces list, and the names an active workspace of it may have.
func sourceScope(src *source.Source) (repoIDs map[string]bool, workspaces []string, err error) {
	names, err := src.WorkspaceNames()
	if err != nil {
		return nil, nil, err
	}

	repoIDs = map[string]bool{}

	for _, n := range names {
		w, err := src.LoadWorkspace(n)
		if err != nil {
			return nil, nil, err
		}

		workspaces = append(workspaces, w.Name, w.Qualified())

		for i := range w.Repos {
			repoIDs[w.Repos[i].ID()] = true
		}
	}

	return repoIDs, workspaces, nil
}

type pullOutput struct {
	From      []string `json:"from"                 yaml:"from"`
	Sources   []string `json:"sources"              yaml:"sources"`
	Paths     []string `json:"paths"                yaml:"paths"`
	Active    string   `json:"active,omitempty"     yaml:"active,omitempty"`
	ScanRoots []string `json:"scan_roots,omitempty" yaml:"scan_roots,omitempty"`
}

func (a *app) statePullCommand() *cobra.Command {
	var f stateFlags

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Merge users/<email>/state.yaml from a source, or from every attached one: local entries win",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.statePull(cmd.Context(), f)
		},
	}
	f.bind(cmd, "source to read from (default: every attached source)")

	return cmd
}

// statePull merges the shared state of one source, or of every source
// attached when it starts, in attach order. Each push holds only what belongs
// to its source, so this laptop is whole again only once every source is read.
func (a *app) statePull(ctx context.Context, f stateFlags) error {
	st, err := a.loadState()
	if err != nil {
		return err
	}

	names := []string{f.source}
	if f.source == "" {
		names = nil
		for _, src := range st.Sources {
			names = append(names, src.Name)
		}
	}

	out := pullOutput{From: []string{}, Sources: []string{}, Paths: []string{}}

	for _, name := range names {
		p, err := a.sharedStatePath(ctx, st, name, f.user)
		if err != nil {
			return err
		}

		if !fileExists(p) {
			if f.source != "" {
				return notFoundf("no shared state at %s", p)
			}

			continue
		}

		if err := a.mergeShared(ctx, st, p, &out); err != nil {
			return err
		}
	}

	if len(out.From) == 0 {
		return notFoundf("no shared state in any attached source")
	}

	if err := a.saveState(st); err != nil {
		return err
	}

	return a.print(out)
}

func (a *app) mergeShared(ctx context.Context, st *state.State, p string, out *pullOutput) error {
	shared, err := state.Load(p)
	if err != nil {
		return err
	}

	merged := st.Merge(shared.Expanded(a.env.Getenv("HOME")))

	// Sources come only from states pushed before pushes were scoped.
	for _, src := range merged.Sources {
		if err := a.attachShared(ctx, src); err != nil {
			return err
		}

		st.Sources = append(st.Sources, src)
		out.Sources = append(out.Sources, src.Name)
	}

	out.From = append(out.From, p)
	out.Paths = append(out.Paths, merged.Paths...)

	if merged.Active != "" {
		out.Active = merged.Active
	}

	if len(merged.ScanRoots) > 0 {
		out.ScanRoots = merged.ScanRoots
	}

	return nil
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
