# 4. Ship to Homebrew as a cask, not a formula

## Decision

goreleaser publishes `Casks/mur.rb` to `bifshteks/homebrew-tap`
(`homebrew_casks`), and the install command is
`brew install --cask bifshteks/tap/mur`. A post-install hook clears the macOS
quarantine flag, because the binary is not notarized. This amends the spec,
which said `Formula/mur.rb`.

## Why

goreleaser deprecated `brews` (formulae) in v2.10 in favour of
`homebrew_casks`, and `goreleaser check` exits non-zero on deprecated
properties. The release workflow follows the latest v2. A formula would have
made every release run on a deprecated path, and would have failed the check
that the ticket requires.

## Not verified

- `goreleaser check` was never run: goreleaser is not installed on the
  laptop this was written on.
- Whether Homebrew on Linux installs the cask was not tested. If it does
  not, Linux users use `go install github.com/bifshteks/mur@latest` or the
  release tarball.
