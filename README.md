# mur

`mur` (multi-repository) is a small CLI that keeps a **map of the git repos you
work with**. For a named **workspace** it tells you which repos the workspace
has, what each one is for, and where each one is cloned on this laptop.

```console
$ mur resolve billing
- query: billing
  workspace: nimbus/billing
  id: git.nimbus.example/nimbus/billing/billing
  url: git@git.nimbus.example:nimbus/billing/billing.git
  path: /Users/me/projects/billing
  cloned: true
  description: All billing logic. We own it and ship it to prod.
  aliases:
    - billing
  langs:
    - go
  status: live
```

It is built for LLM agents first, and works just as well for people:

- **Agents stop guessing paths.** Output is YAML (or `--json`). mur never
  prompts without a terminal, and each exit code tells the agent what to do next.
- **A team shares one map.** Workspace definitions live in an ordinary git
  repo (a *source*). A new teammate or a new laptop is set up with three
  commands.
- **Call repos what you call them.** `mur resolve front` finds
  `acme-web-frontend-nextjs-v2` through its alias.
- **mur doesn't touch your work.** It never commits, never pushes, never pulls
  over a dirty tree, and never moves a clone.

**Contents**

- [Install](#install)
- [Tutorial: your first workspace](#tutorial-your-first-workspace)
- [How-to guides](#how-to-guides)
- [Reference](#reference)
- [Explanation](#explanation)
- [Development](#development)

---

## Install

**Homebrew** (macOS, Linux):

```bash
brew tap bifshteks/mur https://github.com/bifshteks/mur   # once: this repo is its own tap
brew install --cask bifshteks/mur/mur
```

**Go**:

```bash
go install github.com/bifshteks/mur@latest
```

Check that it works with `mur --help`.

**Claude Code skill** (optional). This repo is also a Claude Code marketplace.
Its one plugin ships the `mur:repofind` skill, which tells agents to resolve
repos through mur instead of guessing paths or searching the disk:

```bash
claude plugin marketplace add bifshteks/mur
claude plugin install mur@mur
```

---

## Tutorial: your first workspace

In about five minutes you will build a source with one workspace, add a repo to
it, clone it with `mur sync`, and look it up by its alias. Everything runs in a
scratch directory, so your real mur setup is not touched.

You need `mur`, `git` and network access to github.com.

### 1. Use a scratch mur home

mur keeps its local state in `$MUR_HOME`. Point it at a temporary directory:

```bash
export MUR_HOME="$(mktemp -d)"
mkdir -p ~/mur-tutorial
```

### 2. Create a source

A source is a git repo that holds workspace definitions. Create an empty one:

```bash
mur source init tutorial --path ~/mur-tutorial/source
```

```yaml
name: tutorial
path: /Users/you/mur-tutorial/source
```

The new directory holds `mur.yaml` and an empty `workspaces/`.

### 3. Create a workspace and make it active

```bash
mur workspace create demo --source tutorial --summary "Repos for the mur tutorial"
mur workspace activate demo
```

```yaml
active: tutorial/demo
```

### 4. Add a repo

```bash
mur repo add https://github.com/bifshteks/mur.git \
  --workspace demo \
  --description "The mur CLI itself" \
  --lang go \
  --alias murcli
```

Look it up by its alias:

```bash
mur resolve murcli
```

```yaml
- query: murcli
  workspace: tutorial/demo
  id: github.com/bifshteks/mur
  url: https://github.com/bifshteks/mur.git
  cloned: false
  description: The mur CLI itself
  aliases:
    - murcli
  langs:
    - go
```

mur knows the repo but not where it lives yet: there is no `path`, and
`cloned` is `false`.

### 5. Choose where it goes, and clone it

```bash
mur repo locate murcli ~/mur-tutorial/clones/mur
mur sync
```

```yaml
synced: []
cloned:
  - id: github.com/bifshteks/mur
    path: /Users/you/mur-tutorial/clones/mur
adopted: []
skipped: []
failed: []
needs_path: []
```

### 6. Use it

```bash
mur resolve murcli            # now with path and cloned: true
cd "$(mur resolve murcli --path)"
mur workspace show            # everything an agent reads before it starts work
```

You have built a repo map from scratch and used it. To clean up:

```bash
rm -rf ~/mur-tutorial "$MUR_HOME"; unset MUR_HOME
```

Next, [join your team's source](#join-your-teams-source) or
[build a source for your team](#build-a-source-for-your-team).

---

## How-to guides

### Join your team's source

Your team already keeps a source in git. Attach it, pick your workspace, and
sync:

```bash
mur source add nimbus git@git.nimbus.example:nimbus/services-team/junk/workspace.git
mur workspace list                 # find your team's workspace
mur workspace activate analytics
mur sync
```

By default mur searches `~/projects` for clones you already have. If yours are
elsewhere, run this before `mur sync`:

```bash
mur config set scan_roots ~/code,~/work
```

`mur sync` **adopts** a clone you already have when exactly one clone under the
scan roots has a matching `origin`. In a terminal, it asks where to clone each
repo it cannot place. Repos you skip are listed under `needs_path`; see
[place a repo mur cannot find](#place-a-repo-mur-cannot-find).

### Build a source for your team

```bash
mur source init nimbus --path ~/projects/nimbus-mur
mur workspace create billing --source nimbus --summary "Billing team: metering, invoices, payments"
mur repo add git@git.nimbus.example:nimbus/billing/billing.git \
  --workspace billing --description "All billing logic. We own it and ship it to prod." \
  --lang go --alias billing
```

Then write `workspaces/billing/README.md`. Agents read it through
`mur workspace show` (see [workspace README](#workspace-readme-and-extra-files)).
Publish the source yourself; mur never commits or pushes:

```bash
cd ~/projects/nimbus-mur
git add . && git commit -m "Add billing workspace"
git remote add origin git@git.nimbus.example:nimbus/billing/mur.git && git push -u origin HEAD
```

### Add, change or remove a repo

```bash
mur repo add <url> --workspace <ws> --description "..." [--alias a]... [--lang l]... [--status s]
mur repo remove <alias|url> --workspace <ws>
```

To change a field of an existing repo, edit `workspace.yaml` by hand. mur
keeps your comments, key order and any keys it does not know when it rewrites
the file.

### Mark repos with statuses

Declare statuses in `workspace.yaml` **before** using them; `repo add --status`
refuses one that is not declared:

```yaml
statuses:
  - name: live
    description: Ships to prod. Change freely.
  - name: readonly
    description: Another team's service. Read it, don't change it.
  - name: dead
    description: Superseded. Do not change.
    sync: false          # not cloned or fetched unless `mur sync --all`
```

```bash
mur sync --status live     # sync only live repos
mur sync --all             # also include statuses with sync: false
```

### Keep your clones up to date

```bash
mur sync            # git fetch --prune every clone; never changes a working tree
mur sync --pull     # also fast-forward clones that are clean and on their default branch
mur sync --dry-run  # report only
```

Clones that were not fast-forwarded are listed under `skipped` with the reason.

### Pull your team's latest workspace definitions

```bash
mur source pull            # every source
mur source pull nimbus     # one source
```

### Place a repo mur cannot find

When `mur sync` exits with code `3`, the repos under `needs_path` have no
known location. Record one and sync again:

```bash
mur repo locate front ~/projects/web     # the directory may not exist yet
mur sync
```

The directory must be empty, missing, or already a clone of that repo.

### Move to a new laptop

On the old laptop, save where your repos are into the source. Then commit and
push it yourself:

```bash
mur state push --source nimbus     # writes users/<your git email>/state.yaml
```

On the new laptop:

```bash
mur source add nimbus <source-url>
mur state pull --source nimbus     # local entries win on conflict
mur sync
```

Paths are saved with `~`, so they work under a different home directory.

### Let an AI agent use mur

1. Install the [Claude Code skill](#install), or put the same rule into your
   agent's instructions: *"Before working in a repo, run `mur resolve <name>`
   and work in the returned path. Never guess a path."*
2. The agent runs `mur agent-guide` once per session. It is the full manual,
   generated from the installed binary, so it always matches its version.
3. To pin a workspace for one session, set `MUR_WORKSPACE=billing`.

### Keep a personal or local-only source next to your company one

Sources are independent. Attach as many as you like:

```bash
mur source init personal                        # local only, at $MUR_HOME/sources/personal
mur workspace create pets --source personal
mur workspace activate pets                     # switch back with: mur workspace activate billing
```

`state push` writes only the paths of repos listed in that source's workspaces.
Your company source never learns about your personal repos.

---

## Reference

### Commands

Every command accepts `--json`. Reads take the workspace from `--workspace`,
then `$MUR_WORKSPACE`, then the active workspace. Writes (`repo add`,
`repo remove`) require `--workspace`.

| Command | What it does |
| --- | --- |
| `mur resolve <alias\|basename\|url>...` | What each repo is and where it is cloned. `--path` prints only absolute paths. |
| `mur workspace show [name]` | Summary, README, statuses, repos with paths, extra files. |
| `mur workspace list [--repos]` | Workspaces across all sources. |
| `mur workspace create <name>` | Create a workspace in a source (`--source`, `--summary`). |
| `mur workspace activate <name>` | Make a workspace the default for reads. |
| `mur repo list` | Repos of a workspace with their paths. |
| `mur repo add <url>` | Add a repo. `--description` (required), `--alias`, `--lang`, `--status`. |
| `mur repo remove <alias\|url>` | Remove a repo from a workspace. |
| `mur repo locate <alias\|url> <dir>` | Record where a repo is, or should be, cloned. |
| `mur sync` | Adopt existing clones, clone missing repos, fetch the rest. `--pull`, `--dry-run`, `--status`, `--all`. |
| `mur source init <name>` | Create a source skeleton and attach it (`--path`). |
| `mur source add <name> <git-url\|path>` | Attach an existing checkout, or clone a url (`--path`). |
| `mur source list` / `pull [name]` / `remove <name>` | List, fast-forward, or detach sources. Detaching leaves the files on disk. |
| `mur state push --source <s>` | Write this laptop's state into `users/<email>/state.yaml` of a source. |
| `mur state pull --source <s>` | Merge that file into local state. Local entries win, and missing sources are cloned. |
| `mur config get\|set scan_roots [dirs]` | Where `sync` looks for existing clones (default `~/projects`). |
| `mur agent-guide` | The manual for LLM agents. |

Run `mur <command> --help` for every flag.

### Exit codes

| Code | Meaning | What to do |
| --- | --- | --- |
| `0` | ok | |
| `1` | error | Read stderr. In `sync`, one failing repo does not stop the others. |
| `2` | usage | Bad flags or arguments, or no workspace selected. |
| `3` | needs input | `needs_path` lists repos with no known path. Use `mur repo locate`. |
| `4` | not found or ambiguous | Candidates are printed on stderr. |

### `mur sync` result

| Key | Meaning |
| --- | --- |
| `synced` | Fetched. Shows branch, `dirty`, `ahead`, `behind`. |
| `cloned` | Freshly cloned. |
| `adopted` | An existing clone was found under the scan roots and recorded. It is fetched on the next sync. |
| `skipped` | Not touched, or fetched but not pulled. `reason` says why. |
| `failed` | Something went wrong for this repo only. |
| `needs_path` | No known location; see exit code `3`. |

### Environment

| Variable | Meaning |
| --- | --- |
| `MUR_HOME` | Local state directory. Default `$XDG_CONFIG_HOME/mur`, else `~/.config/mur`. |
| `MUR_WORKSPACE` | Workspace for reads, overriding the active one. |

### Source layout

```
mur.yaml                     # version: 1
workspaces/
  billing/
    workspace.yaml           # statuses and repos
    README.md                # what the team does and its rules; shown to agents
    architecture.md          # any extra files; listed under `files:` by `workspace show`
users/
  me@nimbus.example/
    state.yaml               # written by `mur state push`
```

### `workspace.yaml`

```yaml
summary: Billing team: metering, invoices, payments   # one line; shown in `workspace list`

statuses:                    # optional; a repo's status must be one of these
  - name: live
    description: Ships to prod
    sync: true               # optional, default true; false = sync only with --all
  - name: dead
    description: Superseded. Do not change.
    sync: false

repos:
  - url: git@git.nimbus.example:nimbus/billing/billing.git   # anything `git clone` accepts
    description: All billing logic. We own it and ship it to prod.   # required; keep it short
    aliases: [billing]       # names people and agents use for it
    langs: [go]              # free text; hints where an agent should start reading
    status: live
```

Keep `description` short. An agent reads every repo's description to decide
which repo to work in.

### Workspace README and extra files

`workspaces/<name>/README.md` is printed by `mur workspace show`. Use it to tell
an agent what the team does, how the repos fit together, and the rules for
changing them. Any other file in the directory is listed under `files:`, so the
README can link to longer documents that are opened only on demand.

### Local state (`$MUR_HOME/state.yaml`)

```yaml
active: billing
sources:
  - name: nimbus
    path: /Users/me/.config/mur/sources/nimbus
scan_roots:
  - ~/projects
paths:                       # repo id -> clone path
  git.nimbus.example/nimbus/billing/billing: /Users/me/projects/billing
```

`users/<email>/state.yaml` in a source has the same shape. It holds `active`,
`scan_roots`, and the `paths` of that source's repos, with `~` paths.

### `mur.yaml`

```yaml
version: 1
```

mur refuses a source written with a newer version and asks you to upgrade.

### Repo identity

A repo is identified by its normalized url: `host/path`, with the host
lowercased and the scheme, `user@`, port and a trailing `.git` stripped. So
`git@github.com:bifshteks/mur.git` and `https://github.com/bifshteks/mur` are the
same repo, `github.com/bifshteks/mur`. The path keeps its case. `resolve`
matches an alias, a basename (`mur`), or any form of the url.

---

## Explanation

### The problem mur solves

In a company that uses many repos, every developer and every agent session
keeps asking the same question:

> Where is the repo we call *X*, and what is it for?

People answer it from memory and shell history. Agents answer it by searching
the disk or guessing paths, and they get it wrong. They pick
`svc-analytics-common` when `iam/common` was meant, or edit a stale second
clone. mur turns the answer into data: you describe your repos once, and mur
**resolves** them. It never guesses.

### Why workspaces map to teams

Take a company, *Nimbus Cloud*, with hundreds of repos and dozens of teams. Each
team owns a few repos and only reads some others:

```
security:  iam-access-manager (own), iam-role-manager (own), billing (integrate)
billing:   billing (own), iam-access-manager (integrate), managed-postgres (integrate)
core:      vm-virtualizer (own), iam-access-manager (integrate)
```

A **workspace** is that list for one team, with each repo's purpose and how it
may be treated:

> - *billing* — live, ours, shipped to prod. All billing logic lives here.
> - *iam-access-manager* — read-only. We call its endpoints to check permissions.
> - *openmeter* — upstream project, we don't change it. Read it for ideas.

An agent that reads this (`mur workspace show`) knows where to make changes,
what to only read, and what to leave alone. When a team splits, split the
workspace (`billing-core`, `billing-payments`, …) so each subteam sees only its
own repos.

### Three kinds of data, kept apart

| | Where | Shared? |
| --- | --- | --- |
| **Source**: workspace definitions | a git repo you commit and push | yes, with the team |
| **Local state**: which sources, where clones are | `$MUR_HOME/state.yaml` | no |
| **Saved state**: your paths for one source | `users/<email>/state.yaml` in that source | only if you push it |

Shared definitions and personal paths never mix, and neither do different
sources. A company source never learns about repos from your personal one.
Because mur only writes files and leaves committing to you, it works with
protected branches and with local-only sources.

### Why mur is careful with your clones

mur runs your own `git` binary, so SSH config, credential helpers and
worktrees behave exactly as they do in your terminal. `sync` only *fetches*.
It fast-forwards only on request, and only a clean tree on the default branch.
It adopts an existing clone only when exactly one candidate matches. Without a
terminal, it never waits for input: it finishes what it can and exits `3`. An
agent therefore never blocks, and in-progress work is never moved or
overwritten.

The decisions behind this are recorded in [`docs/adr/`](docs/adr/).

---

## Development

```bash
go test ./...            # acceptance tests in internal/tests drive cli.Run with real git
golangci-lint run ./...
```

`mur agent-guide` is generated from the command tree. Its examples in
`internal/cli/guide/` are checked against real output by the tests.

**Releases**: push a `vX.Y.Z` tag. goreleaser builds darwin/linux × amd64/arm64
and commits `Casks/mur.rb` to `master` of this repo, which is also the Homebrew
tap. The workflow's own `GITHUB_TOKEN` is enough; no extra secret is needed.
