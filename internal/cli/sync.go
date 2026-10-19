package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/repoid"
	"github.com/bifshteks/mur/internal/reposync"
	"github.com/bifshteks/mur/internal/scan"
	"github.com/bifshteks/mur/internal/source"
	"github.com/bifshteks/mur/internal/state"
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

	roots := a.scanRoots(st.ScanRoots)
	for _, r := range roots {
		if !isDir(r) {
			_, _ = fmt.Fprintf(a.env.Stderr, "warning: scan root %s does not exist; mur config set scan_roots changes it\n", r)
		}
	}

	var ask func(context.Context, reposync.Repo, []string) string
	if a.env.IsTTY && !f.dryRun {
		ask = a.askPath(newLineReader(a.env.Stdin), claimedPaths(st.Paths))
	}

	report, placed := reposync.Run(ctx, a.git, syncRepos(w, st, f), &reposync.Options{
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

	if len(placed) > 0 && !f.dryRun {
		if err := a.recordPaths(placed); err != nil {
			return err
		}
	}

	if len(report.NeedsPath) > 0 {
		report.Hint = locateHint(w)
	}

	if err := a.print(report); err != nil {
		return err
	}

	return syncExit(&report)
}

// syncRepos picks the repos this run covers: --status narrows, and statuses
// declared with sync: false are skipped with a reason unless --all.
func syncRepos(w *source.Workspace, st *state.State, f syncFlags) []reposync.Repo {
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

	return repos
}

// recordPaths saves what sync decided. Sync can run for minutes and another
// mur may have changed the state meanwhile, so it applies only its own
// decisions to a fresh read.
func (a *app) recordPaths(placed map[string]string) error {
	fresh, err := a.loadState()
	if err != nil {
		return err
	}

	for id, p := range placed {
		fresh.SetPath(id, p)
	}

	return a.saveState(fresh)
}

func syncExit(report *reposync.Report) error {
	switch {
	case len(report.NeedsPath) > 0:
		return &exitError{code: ExitNeedsPath, err: fmt.Errorf("%d repo(s) need a path", len(report.NeedsPath)), printed: true}
	case len(report.Failed) > 0:
		return &exitError{code: ExitError, err: fmt.Errorf("%d repo(s) failed", len(report.Failed)), printed: true}
	}

	return nil
}

// claimedPaths inverts the recorded paths, so the prompt can refuse to give one
// directory to two repos.
func claimedPaths(paths map[string]string) map[string]string {
	claimed := make(map[string]string, len(paths))
	for id, p := range paths {
		claimed[p] = id
	}

	return claimed
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

// lineReader hands out stdin lines without blocking an interrupt: reading
// happens in its own goroutine, which ends at EOF or with the process.
type lineReader struct {
	lines chan string
}

func newLineReader(in io.Reader) *lineReader {
	lr := &lineReader{lines: make(chan string)}
	br := bufio.NewReader(in)

	go func() {
		defer close(lr.lines)

		for {
			line, err := br.ReadString('\n')
			if line != "" || err == nil {
				lr.lines <- strings.TrimSpace(line)
			}

			if err != nil {
				return
			}
		}
	}()

	return lr
}

// next is the next line; ok is false at EOF or on interrupt.
func (lr *lineReader) next(ctx context.Context) (line string, ok bool) {
	select {
	case <-ctx.Done():
		return "", false
	case l, open := <-lr.lines:
		return l, open
	}
}

// askPath prompts on stderr for where a repo goes: a candidate's number, a
// path that is that repo's clone, a missing or empty directory to clone into,
// or nothing to leave it unplaced. An interrupt or EOF counts as nothing.
func (a *app) askPath(in *lineReader, claimed map[string]string) func(context.Context, reposync.Repo, []string) string {
	return func(ctx context.Context, r reposync.Repo, candidates []string) string {
		name := r.Alias
		if name == "" {
			name = r.ID
		}

		_, _ = fmt.Fprintf(a.env.Stderr, "%s (%s) has no path on this laptop.\n", name, r.URL)
		for i, c := range candidates {
			_, _ = fmt.Fprintf(a.env.Stderr, "  %d) %s\n", i+1, c)
		}

		for {
			_, _ = fmt.Fprint(a.env.Stderr, "Number, path to clone into, or empty to skip: ")

			line, ok := in.next(ctx)
			if !ok || line == "" {
				return ""
			}

			if n, err := strconv.Atoi(line); err == nil {
				if n >= 1 && n <= len(candidates) {
					return candidates[n-1]
				}

				_, _ = fmt.Fprintf(a.env.Stderr, "%d is not one of the %d candidates; type a path or leave empty\n", n, len(candidates))

				continue
			}

			p := a.abs(line)
			if reason := a.unusable(ctx, p, r.ID, claimed); reason != "" {
				_, _ = fmt.Fprintln(a.env.Stderr, reason)

				continue
			}

			claimed[p] = r.ID

			return p
		}
	}
}

// unusable says why a typed path cannot hold the repo, or "" when it can: it
// does not exist yet, is empty, or is a clone of exactly this repo.
func (a *app) unusable(ctx context.Context, p, id string, claimed map[string]string) string {
	if other, ok := claimed[p]; ok && other != id {
		return p + " is already the path of " + other
	}

	entries, err := os.ReadDir(p)
	if os.IsNotExist(err) || (err == nil && len(entries) == 0) {
		return ""
	}

	if a.git.IsRepo(ctx, p) && repoid.Normalize(a.git.Origin(ctx, p)) == id {
		return ""
	}

	return p + " is not a clone of " + id + " and is not empty"
}
