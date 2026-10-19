# 1. The git binary, and a repo's identity is its normalized url

## Decision

mur runs the `git` binary through `os/exec` and never links go-git. A repo is
identified by its normalized `host/path`: lowercase host; scheme, `user@`, port
and a trailing `.git` or `/` stripped; scp `host:path` read as `host/path`;
local paths and `file://` urls kept as absolute paths. The url as written is
stored too, and is what gets cloned.

## Why

SSH config, credential helpers and worktrees then behave exactly as in the
user's terminal. ssh and https spellings of one repo must be one repo, and a
basename is ambiguous (`common` is both `iam/common` and `svc-analytics-common`).
