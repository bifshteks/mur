# 3. Sync never guesses a path and never moves in-progress work

## Decision

- A repo with no recorded path is **adopted** only when exactly one main clone
  under the scan roots has an origin with the same id. Worktrees and submodules
  (`.git` is a file) are not candidates.
- Otherwise a terminal user is asked (a candidate's number, a path, or skip).
  A typed path must not exist yet, be empty, or be a clone of that very repo;
  anything else is refused and asked again. An interrupt ends the questions.
  "Terminal" is a real tty (termios ioctl), never just a character device:
  agent harnesses give mur `/dev/null`. Without a terminal, sync finishes
  everything else and exits 3 with `needs_path`, so an agent never blocks.
- Adopting only records the path. The clone is fetched on the next sync.
- Existing clones get `git fetch --prune` only. `--pull` fast-forwards
  (`merge --ff-only @{upstream}` after the fetch) only a clean tree on
  `origin/HEAD`'s branch. Anything else lands in `skipped` with
  `fetched, not pulled: <reason>`.
- Up to 8 repos sync at once, each under a 5 minute deadline. One repo
  failing is reported in `failed` and costs only itself. Exit 1 if any failed
  and nothing needs a path.
- A cancelled git gets SIGINT, so it cleans up after itself, and mur removes
  whatever a failed clone left. The next sync then just clones again.
- Every url and path goes to git after `--`. Sources are shared, so a url can
  never be read as a git option.
- Sync records the paths it decided onto a fresh read of the state, so a
  `config set` or `repo locate` made while it ran survives. The state file is
  written to a temp file and renamed into place.

## Why

It makes migration free on a laptop that already has its clones, keeps
`~/.auditlogs/wt` worktrees and feature branches safe, and makes the agent's
next step explicit in the exit code.
