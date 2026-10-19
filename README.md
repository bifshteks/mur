# mur

`mur` (multi-repository) answers, for a named **workspace**: which repos are in
it, what each one is for, and where each one is cloned on this laptop. Its main
reader is an LLM harness: output is YAML (or `--json`), it never prompts without
a terminal, and its exit codes say what to do next.

```bash
brew install --cask bifshteks/tap/mur   # or: go install github.com/bifshteks/mur@latest
```

## Quick start

```bash
mur source add mws git@git.mws-team.ru:mws/security-services/junk/workspace.git
mur workspace activate auditlogs
mur sync                            # adopts existing clones, clones the rest
mur resolve collector logrotate     # urls, paths, descriptions, langs, status, notes
cd "$(mur resolve collector --path)"
```

On a new laptop, `mur source add` → `mur state pull --source mws` → `mur sync`
restores everything, asking only about repos it cannot place.

Agents: run `mur agent-guide` once per session. It is generated from the command
tree, and its examples are checked against real output by the tests.

## Concepts

- **source**: a git checkout holding `mur.yaml` and `workspaces/`. Several can be
  attached. A source with no remote is local-only.
- **workspace**: `workspaces/<name>/workspace.yaml` plus `README.md`. Other files
  there are listed by `mur workspace show`, never parsed.
- **repo id**: the normalized `host/path` of a url, so ssh and https spellings are
  one repo. Aliases are unique within a workspace only.
- **local state**: `$MUR_HOME/state.yaml` (default `~/.config/mur`): sources,
  active workspace, scan roots, and repo id → path. Personal and never shared
  unless you `mur state push`.

mur only writes files. Committing and pushing a source is plain git, yours to do.

```yaml
# workspaces/auditlogs/workspace.yaml
summary: Audit events of MWS Cloud.
statuses:                 # optional; the team's own enum
  - name: live
    description: Ships, deployed or used.
  - name: dead
    description: Superseded. Do not change.
    sync: false           # sync skips it unless --all
repos:
  - url: git@git.mws-team.ru:mws/security-services/sec-audit-collector.git
    description: Reads k8s node files and exports them.
    aliases:
      - collector
    langs:
      - go
    status: live
    notes: Never log event bodies.
```

## Exit codes

`0` ok · `1` error · `2` usage · `3` needs input (`needs_path`) · `4` not found or
ambiguous (candidates on stderr).

## Development

```bash
go test ./...          # acceptance tests in internal/tests drive cli.Run with real git
golangci-lint run ./...
```

Releases: push a `vX.Y.Z` tag; goreleaser builds darwin/linux × amd64/arm64 and
writes `Casks/mur.rb` to `bifshteks/homebrew-tap` (secret
`HOMEBREW_TAP_GITHUB_TOKEN`). Decisions are in [`docs/adr/`](docs/adr/).
