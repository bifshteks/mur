# mur

`mur` (multi-repository) answers, for a named **workspace**: which repos are in
it, what each one is for, and where each one is cloned on this laptop. Its main
reader is an LLM harness: output is YAML (or `--json`), it never prompts without
a terminal, and its exit codes say what to do next.

## CLI Installation

CLI may by installed via different managers:

## brew

```bash
brew tap bifshteks/mur https://github.com/bifshteks/mur   # once: this repo is its own tap
brew install --cask bifshteks/mur/mur
```

## golang

```bash
go install github.com/bifshteks/mur@latest
```

## Agent skill

This repo is also a Claude Code marketplace. Its one plugin ships the
`mur:repofind` skill, which tells agents to resolve repos through `mur` instead
of guessing paths or searching the disk:

```bash
claude plugin marketplace add bifshteks/mur
claude plugin install mur@mur
```

The skill only says when to reach for `mur`; the manual is `mur agent-guide`,
so the skill does not drift from the installed binary.

The skill is released with the CLI under the same `vX.Y.Z` tag. Each updates on
its own (`brew upgrade mur`, `claude plugin update mur@mur`), so the two installed
versions may differ, and that is fine.

## Quick start

```bash
# ensure mur is installed
mur --help
# add your company existing mur configuration (aka source)
mur source add nimbus git@git.nimbus.example:nimbus/services-team/junk/workspace.git
# activate workspace 'analytics', which mur finds in the source
mur workspace activate analytics
# adopt existing clones, clone the rest
mur sync
# get urls, paths, descriptions, langs, status, notes by searching repository names/aliases
mur resolve collector logrotate
# cd into a concrete resolved repository
cd "$(mur resolve collector --path)"
```

On a new laptop, `mur source add` → `mur state pull --source nimbus` → `mur sync`
restores everything, asking only about repos it cannot place.

Agents: run `mur agent-guide` once per session. It is generated from the command
tree, and its examples are checked against real output by the tests.


## Core idea

Everything `mur` does is just answering a question

> Where is a repo called <qwerty> lives and what is a brief description of it?

<qwerty> here is some exact name, alias or informal name of a repository.
Thus, instead of always using full path or exact repository name (e.g. "my-long-name-of-some-complex-repo-i-want-to-keep-format-to-avoid-collision"),
you use the one you use in every day live (e.g. "scratcher").

You give knowledge about your repositories and their description to `mur` - it does not know anything until you explicitly configure it.
`mur` does not guess - it resolves.

`mur`'s primary focus is companies using multi-repo approach with multiple teams, each working on some subset of the repos.

For example, you have a company called "Nimbus Cloud".
It has hundreds of repositories and dozens of teams.

```
Repositories:
- iam-access-manager
- iam-role-manager
- billing
- vm-virualizizer
- customer-support
- managed-postgres
- managed-k8s
- ...
```

Each team has some repository they maintain, and some repository they sometimes need access to
(for example, to make proper integration with or find out the behaviour nuances of the external system the use)
```
## Teams:

security:
- iam-access-manager    <-- maintain
- iam-role-manager      <-- maintain
- billing               <-- integrate with

billing:
- billing               <-- maintain
- iam-access-manager    <-- intergrate with
- managed-postgres      <-- intergrate with

core:
- vm-virualizizer       <-- maintain
- iam-access-manager    <-- intergrate with

support:
- customer-support      <- maintain

managed databases:
- managed-postgres      <-- maintain
- managed-k8s           <-- maintain
- iam-access-manager    <-- intergrate with

...
```

In order to power up their developers experience using LLM to work around the company codebase, Nimbus can use `mur`.
A repository with `mur` configuration is created within Nimbus isolated infrastructure environment (e.g. on-premes GitLab).

The `mur` config repository contains **Workspaces** - per team description of repositories used by a team in their work.
Within each **workspace** each team describes list of repositories and how they should treated,
e.g. for "billing" workspace it would be something like (example in prose):

> * We use "billing" repository - this is live code. We maintain it and push to prod. This is where all billing logic lives.
> * We also use "iam-access-manager" repository. We can only read this repo. We call this service's endpoints in order to check permissions.
> * We also use "openmeter" repository. We do not maintain it. We sometimes read this repo in order to get inspired when thinking about billing features or architecture approaches.



## Concepts

### Source
A git checkout holding `mur.yaml` and `workspaces/`.
Several can be attached. A local-only source (a source with no remote) is also supported.

`mur` can be configured to search from different sources.
For example, you may have 4:
* main company's source - your primary job.
* side company's source - from company you work pro-bono.
* personal source - you own pet-projects you maintain in the late nights after job.
* local source - your personal MVP projects you do not dare even publish to github yet.


### Workspace
A named list of repositories within a source.

Think of a **workspace** as a "name of a team".
For personal **source** it does not make sense to create more than one workspace.
For a repositories where you touch only part of them - it make sense, e.g.:
* frontend
* backend

You create a workspace when you need another set of repositories different from existing workspaces' ones.
E.g. you were working at billing, and all billing-team worked on all of 30 repositories - everyone changes the code in every of 30 repos.
Thus, you had workspace
* billing

But some day you decided to split the team into subteams. Thus, you may split workspaces accordingly:

* billing-core
* billing-metering
* billing-payments
* billing-customer

Thus, each of subteams known and cares only about the repos described in *theirs* workspace.


### Repository
A regular git repository.

### Local state
`mur` needs some data stored so it does not recalculate it on every run.
The data includes:
* a map of `host/path` to full path on local disk

Local state can also be stored within a **Source** (`mur state push`), so if you change your laptop you get the repositories path
the same as where on previous laptop - all your preferences are kept.

## How to

1. [Optional] Change where will `mur` will search for existing repositories from
```
mur config set scan_roots ~/Projects
```

2. Initialize a mur configuration source (i.e. create new one)

```
mkdir path/to/newfolder
mur source init my-source-name --path path/to/newfolder
```

This will:
* create a skeleton of a cur configuration source named "my-source-name" in specified directory (see `mur source init --help` for more info)
* register it in local mur state, so mur knows about this source

3. Add a workspace
```
mur workspace create backend --source my-source-name
```

4. Activate the workspace
```
mur workspace activate backend
```

This will:
* add a skeleton of a workspace within a source we just created

5. Add a repository to a workspace
```
mur repo add git@github.com:bifshteks/mur.git \
  --workspace backend \
  --description "Some brief description of the repository"
  --lang golang
  --alias aboba
  --alias pivo
```

This will:
* validate url (any form we can "git clone <url>" is supported)
* add a repository "git@github.com:bifshteks/mur.git" to a workspace we created with given description and lang.
  The repository will be searchable by using either of: "aboba", "pivo"

6. [Optional] Validate intermediate result
```
mur resolve aboba
```

This must return the description of the repository you added, but with no "path" key and:
```
...
  cloned: false
...
```

7. Sync repositories
```
mur sync
```

This:
* ask for a local path to clone repo into
* clones all the missing repos

8. [Optional] Validate result
```
mur resolve aboba
```

This must return the description of the repository you added, but now with:
```
...
  path: <local path you specified on prevous step>
  cloned: true
...
```

9. [Optional] Save you preferences inside this configuration
```
mur state push --source my-source-name
```

This will:
* save you resolved paths under `./users/<git login>/state.yaml`.

9. [Optional] Publish the source, so your teammates can use it too
Git init/add/commit/push the source you just created ("path/to/newfolder" in our example).

> `mur` only writes files. Committing and pushing a source is plain git, yours to do.


### Configuration

A source layout is:
```
users/
    some@git.login/          <-- personal configs of some@git.login user that they decided to save in this source
        state.yaml           <-- state of their "repo:localpath" map
    another@git.login/
        state.yaml
workspaces/
    some-workspace-name/
        workspace.yaml       <-- workspace configuration: status list, repo list, their descriptions/aliases/etc
        README.md            <-- general explanation of this workspace. This will be given to an LLM so it knows "what is your team is doing, what are main rules, what is context of your work". May links any other files in this folder
        custom.file          <-- any other files you may want to add here. README.md may link to it, so LLM can reach it.
    another-workspace-name/
        workspace.yaml
        README.md
```


#### Reading and manual configuration


##### Workspace (`workspaces/some_workspace/workspace.yaml`)
```
# Short description of this workspace.
# Used by you/yourTeammates/LLM in find out whether this is the right workspace to use.
summary: ""

# [Optional] List of "tags" you would like to mark repositories in "repos" with.
# You can add any number of statuses you want.
statuses:
    # name of a tag
  - name: live

    # what this tag mean
    description: Ships, deployed or used

    # [Optional, default - true]
    # If false - repo with this status is not cloned unless "--all"
    sync: true

  - name: dead
    description: Superseded. Do not change.
    sync: false


# List of repositories that a developer using this workspace touches during their work.
repos:
    # Url in any form a "git clone" accepts it: ssh, https, etc.
  - url: git@github.com/projectnessie/nessie.git

    # [Required] Purpose of this is to give to LLM some context for reasoning whether this is the right repository
    # to make changes in. Should not be large - LLM receives all of repositories list and all of their descriptions.
    description: mydescription

    # [Optional] List of nicknames.
    # When in real life or LLM session you do not really call a repository by its full name (e.g. "clean-architecture-swiftui")
    # and use some shortened alias - add it here, so mur (and LLM using mur) can find a repo by its alias
    aliases:
      - aboba

    # [Optional] Programming languagues used by this repository
    # Purpose of this - is to help LLM to search within this repo.
    # No validation on this in mur - you can write in free form.
    # E.g. if repository's lang is "go", LLM is likely to start with "go.mod" in the root directory or "cmd/" folder.
    langs:
      - go
      - codegen

    # [Optional] "Tag" you mark this repository with
    status: live

  - url: git@github.com:nalexn/clean-architecture-swiftui.git
    description: mydescription
    aliases:
      - pivo
```

##### Personal state (`users/some@git.login/state.yaml`)
```
# name of a workspace currently active.
# NOTE: This is not updated without "mur state push"
active: backend

# list of folders mur searches for repositories in order to find out whether their clone is already exists or we need to clone it.
scan_roots:
  - ~/projects

# a mapping of "repo's host/url" to localpath where a repository must be cloned
paths:
  github.com/nalexn/clean-architecture-swiftui: ~/projects/nalexn/clean-architecture-swiftui
  github.com/projectnessie/nessie: ~/projects/nessie
```

##### Root (`mur.yaml`)
```
# Just version of mur.
# When runs, mur checks what version the configuration was written with.
# If source is written with newer version - asks to upgrade.
version: 1
```

## Exit codes

`0` ok · `1` error · `2` usage · `3` needs input (`needs_path`) · `4` not found or
ambiguous (candidates on stderr).

## Reporting a problem

Include both versions; each is the git tag of the source you ran:

```bash
mur --version          # the CLI
claude plugin list     # the skill: the "Version" under mur@mur
```

## Development

```bash
go test ./...          # acceptance tests in internal/tests drive cli.Run with real git
golangci-lint run ./...
```

Releases: push a `vX.Y.Z` tag; goreleaser builds darwin/linux × amd64/arm64 and
commits `Casks/mur.rb` to `master` of this repo, which is also the Homebrew tap.
No extra secret: the workflow's own `GITHUB_TOKEN` writes it. A second job then
pins the plugin in `.claude-plugin/marketplace.json` to the tag
(`tools/pinplugin`), so `master` always offers the last released skill. To try
unreleased skill changes, run `claude --plugin-dir .` from a checkout.
Decisions are in [`docs/adr/`](docs/adr/).
