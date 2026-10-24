# 2. Shared definitions in sources, personal state in MUR_HOME

## Decision

Workspace definitions live in sources (git checkouts), and mur only edits files
there: it never commits or pushes, except `source add` (clone) and
`source pull`. Where things are on this laptop lives in `$MUR_HOME/state.yaml`.
`state push` copies the part of it that belongs to a source — paths of repos
its workspaces list, the active workspace if it is one of them, scan roots —
into that source under `users/<email>/` with `~` paths. The list of sources and
paths of other sources' repos never leave the laptop. `state pull` merges one
source's file, or every attached source's, with local entries winning.

`workspace.yaml` is edited through yaml.v3 nodes, so comments, key order and
unknown keys survive a rewrite. Every node mur creates is block style.

## Why

Shared and personal data must not mix, nor data of different sources: a
company source must not learn about personal projects registered through a
personal one, and protected branches or local-only
sources rule out auto-commit. Node round-trips let an older mur edit a newer
mur's file without dropping what it does not understand. `mur.yaml` carries
`version: 1`, and a higher version is refused with "upgrade mur".
