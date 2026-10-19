package cli

import (
	"bufio"
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/reposync"
	"github.com/bifshteks/mur/internal/scan"
)

const (
	syncConcurrency = 8
	syncTimeout     = 5 * time.Minute
)

type syncFlags struct {
	workspace string
	all       bool
	statuses  []string
	pull      bool
	dryRun    bool
}

func (a *app) syncCommand() *cobra.Command {
	var f syncFlags

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Adopt existing clones, clone missing repos, fetch the rest; never changes a working tree",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return a.sync(cmd.Context(), f)
		},
	}
	cmd.Flags().StringVar(&f.workspace, "workspace", "", "workspace to sync (default $MUR_WORKSPACE, then the active one)")
	cmd.Flags().BoolVar(&f.all, "all", false, "include statuses declared with sync: false")
	cmd.Flags().StringArrayVar(&f.statuses, "status", nil, "only repos with this status; repeatable")
	cmd.Flags().BoolVar(&f.pull, "pull", false, "also fast-forward clean clones that are on their default branch")
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "report what would happen; clone, fetch and record nothing")

	return cmd
}

func (a *app) sync(ctx context.Context, f syncFlags) error {
	w, st, err := a.readWorkspace(f.workspace)
	if err != nil {
		return err
	}

	repos := make([]reposync.Repo, 0, len(w.Repos))

	for i := range w.Repos {
		r := &w.Repos[i]
		if len(f.statuses) > 0 && !slices.Contains(f.statuses, r.Status) {
			continue
		}

		sr := reposync.Repo{ID: r.ID(), URL: r.URL, Alias: r.Alias(), Path: st.Paths[r.ID()]}
		if len(f.statuses) == 0 && !f.all && !w.Syncs(r.Status) {
			sr.Skip = "status " + r.Status + " has sync false"
		}

		repos = append(repos, sr)
	}

	roots := a.scanRoots(st.ScanRoots)
	for _, r := range roots {
		if !isDir(r) {
			_, _ = fmt.Fprintf(a.env.Stderr, "warning: scan root %s does not exist; mur config set scan_roots changes it\n", r)
		}
	}

	var ask func(reposync.Repo, []string) string
	if a.env.IsTTY && !f.dryRun {
		ask = a.askPath(bufio.NewReader(a.env.Stdin))
	}

	report, placed := reposync.Run(ctx, a.git, repos, &reposync.Options{
		ScanRoots:   roots,
		Concurrency: syncConcurrency,
		Timeout:     syncTimeout,
		Clones: func(ctx context.Context, roots []string) map[string][]string {
			return scan.Clones(ctx, a.git, roots)
		},
		Ask:    ask,
		Pull:   f.pull,
		DryRun: f.dryRun,
	})

	for id, p := range placed {
		st.SetPath(id, p)
	}

	if len(placed) > 0 && !f.dryRun {
		if err := a.saveState(st); err != nil {
			return err
		}
	}

	if len(report.NeedsPath) > 0 {
		report.Hint = locateHint(w)
	}

	if err := a.print(report); err != nil {
		return err
	}

	switch {
	case len(report.NeedsPath) > 0:
		return &exitError{code: ExitNeedsPath, err: fmt.Errorf("%d repo(s) need a path", len(report.NeedsPath)), printed: true}
	case len(report.Failed) > 0:
		return &exitError{code: ExitError, err: fmt.Errorf("%d repo(s) failed", len(report.Failed)), printed: true}
	}

	return nil
}

// scanRoots are where sync looks for clones to adopt, ~ expanded.
func (a *app) scanRoots(configured []string) []string {
	if len(configured) == 0 {
		configured = defaultScanRoots
	}

	roots := make([]string, 0, len(configured))
	for _, r := range configured {
		roots = append(roots, a.abs(r))
	}

	return roots
}

// askPath prompts on stderr for where a repo goes: a candidate's number, a
// path (cloned there if it does not exist), or nothing to leave it unplaced.
func (a *app) askPath(in *bufio.Reader) func(reposync.Repo, []string) string {
	return func(r reposync.Repo, candidates []string) string {
		name := r.Alias
		if name == "" {
			name = r.ID
		}

		_, _ = fmt.Fprintf(a.env.Stderr, "%s (%s) has no path on this laptop.\n", name, r.URL)
		for i, c := range candidates {
			_, _ = fmt.Fprintf(a.env.Stderr, "  %d) %s\n", i+1, c)
		}

		_, _ = fmt.Fprint(a.env.Stderr, "Number, path to clone into, or empty to skip: ")

		line, _ := in.ReadString('\n')
		line = strings.TrimSpace(line)

		if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(candidates) {
			return candidates[n-1]
		}

		if line == "" {
			return ""
		}

		return a.abs(line)
	}
}
