# 5. The skill is released with the CLI, from the same tag

## Decision

A tag `vX.Y.Z` releases the CLI and the `mur` Claude Code plugin together, under
the same number. After goreleaser has published the binaries and committed
`Casks/mur.rb`, the release workflow commits `.claude-plugin/marketplace.json`
to `master` with the plugin entry pinned to that tag:

    "version": "X.Y.Z",
    "source": {"source": "url", "url": "https://github.com/bifshteks/mur.git",
               "ref": "vX.Y.Z", "sha": "<tag commit>"}

`tools/pinplugin` does the rewrite, in a `pin-plugin` job that runs after
goreleaser on `master` as goreleaser left it. `plugin.json` carries no version.
No version is ever edited by hand.

## Why

When someone reports a problem, `mur --version` and `claude plugin list` each
name a tag, and that tag is the exact source they ran. Before this, the plugin's
source was `"./"`: it installed whatever `master` held, and Claude Code reported
it as a commit hash unrelated to any release. A hand-kept `version` would have
been worse. Claude Code caches and updates by that string, so a forgotten bump
both hides new commits under an old number and stops updates.

## Consequences

- The installed CLI and skill versions can differ (`brew upgrade` without
  `claude plugin update`). That is allowed. The skill holds no command examples
  and sends the agent to `mur agent-guide`, so any skill version works with any
  CLI.
- A skill-only fix ships as a patch release, with an unchanged CLI rebuild.
- `master`'s marketplace always points at the last release. To try unreleased
  skill changes, use `claude --plugin-dir <checkout>`.
- Each release adds a second bot commit to `master`, next to `Cask mur vX.Y.Z`.
- If the push fails, re-run only the `pin-plugin` job; the release itself
  already exists. `pinplugin` never moves the pin to a lower version, so a re-run
  of an old release, or a hotfix tag on an older line, leaves the newer pin in
  place. Prerelease tags (`v1.2.3-rc.1`) release the CLI and skip the pin.
- A test pins the repo's own marketplace.json, so a manifest field `pinplugin`
  does not model fails `go test` before goreleaser publishes anything, not
  after.
- `claude plugin validate` is not run in CI: it would install the Claude Code CLI
  on every release. The pinned shape was validated once by hand when this was
  written; a change to what `pinplugin` writes needs the same check.
- The tag scheme stays `vX.Y.Z`. `claude plugin tag`'s `{name}--v{version}` is
  for repos holding several plugins.
