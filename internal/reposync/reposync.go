package reposync

import (
	"context"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bifshteks/mur/internal/git"
)

// Repo is one repo to sync: its definition and where this laptop keeps it.
type Repo struct {
	ID    string
	URL   string
	Alias string
	Path  string
	// Skip, when set, is why this repo is left alone this run.
	Skip string
}

type Options struct {
	ScanRoots   []string
	Concurrency int
	// Timeout bounds one repo's clone or fetch, so a hung remote costs only itself.
	Timeout time.Duration
	// Clones finds existing clones under the scan roots, by repo id.
	Clones func(ctx context.Context, roots []string) map[string][]string
	// Ask, when set, asks a person where a repo goes: a path, or "" to skip.
	Ask    func(r Repo, candidates []string) string
	Pull   bool
	DryRun bool
}

type Report struct {
	Synced    []Synced    `json:"synced"     yaml:"synced"`
	Cloned    []Placed    `json:"cloned"     yaml:"cloned"`
	Adopted   []Placed    `json:"adopted"    yaml:"adopted"`
	Skipped   []Skipped   `json:"skipped"    yaml:"skipped"`
	Failed    []Failed    `json:"failed"     yaml:"failed"`
	NeedsPath []NeedsPath `json:"needs_path" yaml:"needs_path"`
	Hint      string      `json:"hint,omitempty" yaml:"hint,omitempty"`
}

type Synced struct {
	ID     string `json:"id"               yaml:"id"`
	Branch string `json:"branch"           yaml:"branch"`
	Dirty  bool   `json:"dirty"            yaml:"dirty"`
	Ahead  *int   `json:"ahead,omitempty"  yaml:"ahead,omitempty"`
	Behind *int   `json:"behind,omitempty" yaml:"behind,omitempty"`
}

type Placed struct {
	ID   string `json:"id"   yaml:"id"`
	Path string `json:"path" yaml:"path"`
}

type Skipped struct {
	ID     string `json:"id"     yaml:"id"`
	Reason string `json:"reason" yaml:"reason"`
}

type Failed struct {
	ID    string `json:"id"    yaml:"id"`
	Error string `json:"error" yaml:"error"`
}

type NeedsPath struct {
	ID         string   `json:"id"              yaml:"id"`
	Alias      string   `json:"alias,omitempty" yaml:"alias,omitempty"`
	Candidates []string `json:"candidates"      yaml:"candidates"`
}

func newReport() Report {
	return Report{
		Synced: []Synced{}, Cloned: []Placed{}, Adopted: []Placed{},
		Skipped: []Skipped{}, Failed: []Failed{}, NeedsPath: []NeedsPath{},
	}
}

// Run places every repo it can, then clones or fetches the placed ones. It
// returns the report and the paths it decided, for the caller to record.
func Run(ctx context.Context, g git.Git, repos []Repo, opt Options) (Report, map[string]string) {
	report := newReport()
	placed := map[string]string{}

	var (
		unplaced []Repo
		work     []Repo
	)

	for _, r := range repos {
		if r.Skip != "" {
			report.Skipped = append(report.Skipped, Skipped{ID: r.ID, Reason: r.Skip})

			continue
		}

		if r.Path == "" {
			unplaced = append(unplaced, r)
		} else {
			work = append(work, r)
		}
	}

	if len(unplaced) > 0 {
		clones := opt.Clones(ctx, opt.ScanRoots)

		for _, r := range unplaced {
			candidates := clones[r.ID]
			if len(candidates) == 1 {
				placed[r.ID] = candidates[0]
				report.Adopted = append(report.Adopted, Placed{ID: r.ID, Path: candidates[0]})

				continue
			}

			if opt.Ask != nil {
				if p := opt.Ask(r, candidates); p != "" {
					placed[r.ID] = p

					if isDir(p) {
						report.Adopted = append(report.Adopted, Placed{ID: r.ID, Path: p})
					} else {
						r.Path = p
						work = append(work, r)
					}

					continue
				}
			}

			report.NeedsPath = append(report.NeedsPath, NeedsPath{
				ID: r.ID, Alias: r.Alias, Candidates: append([]string{}, candidates...),
			})
		}
	}

	results := make([]outcome, len(work))
	sem := make(chan struct{}, max(opt.Concurrency, 1))

	var wg sync.WaitGroup

	for i, r := range work {
		wg.Add(1)

		go func() {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			rctx, cancel := context.WithTimeout(ctx, opt.Timeout)
			defer cancel()

			results[i] = syncOne(rctx, g, r, opt)
		}()
	}

	wg.Wait()

	for _, o := range results {
		o.addTo(&report)
	}

	return report, placed
}

// outcome is what happened to one placed repo; exactly one field is set.
type outcome struct {
	synced  *Synced
	cloned  *Placed
	skipped *Skipped
	failed  *Failed
}

func (o outcome) addTo(r *Report) {
	switch {
	case o.synced != nil:
		r.Synced = append(r.Synced, *o.synced)
	case o.cloned != nil:
		r.Cloned = append(r.Cloned, *o.cloned)
	case o.skipped != nil:
		r.Skipped = append(r.Skipped, *o.skipped)
	case o.failed != nil:
		r.Failed = append(r.Failed, *o.failed)
	}
}

func syncOne(ctx context.Context, g git.Git, r Repo, opt Options) outcome {
	failed := func(err error) outcome { return outcome{failed: &Failed{ID: r.ID, Error: err.Error()}} }

	if _, err := os.Stat(r.Path); os.IsNotExist(err) {
		if !opt.DryRun {
			if _, err := g.Run(ctx, "", "clone", "-q", r.URL, r.Path); err != nil {
				return failed(err)
			}
		}

		return outcome{cloned: &Placed{ID: r.ID, Path: r.Path}}
	}

	if !opt.DryRun {
		if _, err := g.Run(ctx, r.Path, "fetch", "-q", "--prune"); err != nil {
			return failed(err)
		}
	}

	s, err := where(ctx, g, r)
	if err != nil {
		return failed(err)
	}

	if !opt.Pull || opt.DryRun {
		return outcome{synced: &s}
	}

	if reason := pullBlocker(ctx, g, r, s); reason != "" {
		return outcome{skipped: &Skipped{ID: r.ID, Reason: "fetched, not pulled: " + reason}}
	}

	if _, err := g.Run(ctx, r.Path, "merge", "-q", "--ff-only", "@{upstream}"); err != nil {
		return failed(err)
	}

	if s, err = where(ctx, g, r); err != nil {
		return failed(err)
	}

	return outcome{synced: &s}
}

// pullBlocker says why a fast-forward would be unsafe: only a clean tree on
// the remote's default branch is touched, so in-progress work never moves.
func pullBlocker(ctx context.Context, g git.Git, r Repo, s Synced) string {
	if s.Dirty {
		return "working tree dirty"
	}

	head, err := g.Run(ctx, r.Path, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "origin/HEAD is not set"
	}

	if def := strings.TrimPrefix(head, "origin/"); s.Branch != def {
		return "on " + s.Branch + ", default is " + def
	}

	return ""
}

func isDir(p string) bool {
	fi, err := os.Stat(p)

	return err == nil && fi.IsDir()
}

// where reports a clone's branch, dirtiness and distance from its upstream.
func where(ctx context.Context, g git.Git, r Repo) (Synced, error) {
	s := Synced{ID: r.ID}

	branch, err := g.Run(ctx, r.Path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return s, err
	}

	s.Branch = branch

	status, err := g.Run(ctx, r.Path, "status", "--porcelain")
	if err != nil {
		return s, err
	}

	s.Dirty = status != ""

	// No upstream is normal for a local branch; ahead/behind are then omitted.
	if counts, err := g.Run(ctx, r.Path, "rev-list", "--left-right", "--count", "HEAD...@{upstream}"); err == nil {
		if a, b, ok := strings.Cut(counts, "\t"); ok {
			ahead, errA := strconv.Atoi(a)
			behind, errB := strconv.Atoi(b)

			if errA == nil && errB == nil {
				s.Ahead, s.Behind = &ahead, &behind
			}
		}
	}

	return s, nil
}
