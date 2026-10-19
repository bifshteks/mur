# 3. Sync never guesses a path and never moves in-progress work

## Decision

- A repo with no recorded path is **adopted** only when exactly one main clone
  under the scan roots has an origin with the same id. Worktrees and submodules
  (`.git` is a file) are not candidates.
- Otherwise a terminal user is asked (a candidate's number, a path, or skip).
  Without a terminal, sync finishes everything else and exits 3 with
  `needs_path`, so an agent never blocks.
- Adopting only records the path. The clone is fetched on the next sync.
- Existing clones get `git fetch --prune` only. `--pull` fast-forwards
  (`merge --ff-only @{upstream}` after the fetch) only a clean tree on
  `origin/HEAD`'s branch. Anything else lands in `skipped` with
  `fetched, not pulled: <reason>`.
- Up to 8 repos sync at once, each under a 5 minute deadline. One repo
  failing is reported in `failed` and costs only itself. Exit 1 if any failed
  and nothing needs a path.

## Why

It makes migration free on a laptop that already has its clones, keeps
`~/.analytics/wt` worktrees and feature branches safe, and makes the agent's
next step explicit in the exit code.
