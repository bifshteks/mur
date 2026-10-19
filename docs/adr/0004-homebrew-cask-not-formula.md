# 4. Ship to Homebrew as a cask, from a tap inside this repo

## Decision

goreleaser publishes `Casks/mur.rb` (`homebrew_casks`) to `master` of
`bifshteks/mur` itself, which doubles as the Homebrew tap. Installing is:

    brew tap bifshteks/mur https://github.com/bifshteks/mur
    brew install --cask bifshteks/mur/mur
 A post-install hook clears the macOS
quarantine flag, because the binary is not notarized. This amends the spec,
which said `Formula/mur.rb`.

## Why a cask

goreleaser deprecated `brews` (formulae) in v2.10 in favour of
`homebrew_casks`, and `goreleaser check` exits non-zero on deprecated
properties. The release workflow follows the latest v2. A formula would have
made every release run on a deprecated path, and would have failed the check
that the ticket requires.

## Why the tap is this repo

A separate `bifshteks/homebrew-tap` would give a one-line install, because
`brew install bifshteks/tap/mur` finds `github.com/bifshteks/homebrew-tap` by
name. But it costs a second repo and a personal access token secret, since the
workflow's `GITHUB_TOKEN` cannot write to another repo. For a tool installed
once per laptop, the in-repo tap trades that for a one-time `brew tap … <url>`.
The price is one bot commit (`Cask mur vX.Y.Z`) on `master` per release. If
`master` is ever protected, goreleaser needs a `branch` it may push to.

## Not verified

- `goreleaser check` was never run: goreleaser is not installed on the
  laptop this was written on.
- Whether Homebrew on Linux installs the cask was not tested. If it does
  not, Linux users use `go install github.com/bifshteks/mur@latest` or the
  release tarball.

## Watch for

goreleaser's docs mark `hooks.post.install` as deprecated in favour of
`install_steps`. That change is not released yet; it is due in v2.19. The
release workflow follows `~> v2`, so once v2.19 ships, move the quarantine
hook to `post.install_steps` before `goreleaser check` starts failing on it.
