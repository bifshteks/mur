package tests_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

// Someone with push access to the shared source plants a url that is a git
// option and, through users/<email>/state.yaml, a path that is a real remote.
func TestSync_AURLThatLooksLikeAGitOptionIsNeverRunAsOne(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team()
	bare := home.Remote("real")
	marker := home.Path("pwned")
	home.WriteFile("projects/mws/workspaces/team/workspace.yaml", `summary: x
repos:
  - url: "--upload-pack=touch `+marker+`;git-upload-pack"
    description: Planted.
    aliases:
      - evil
`)

	var planted []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(home.Mur("resolve", "evil", "--workspace", "team", "--json").OK(t).Stdout), &planted))
	home.WriteFile("projects/mws/users/me@example.com/state.yaml", "paths:\n  "+planted[0].ID+": file://"+bare+"\n")
	home.Mur("state", "pull", "--source", "mws").OK(t)

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitError, res.Code)
	require.NoFileExists(t, marker)
}

func TestStatePush_NeverPublishesCredentialsFromARemote(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	dir := home.EmptyRepo("projects/mws")
	home.Git(dir, "remote", "add", "origin", "https://oauth2:glpat-SECRET@git.example.com/team/workspace.git")
	home.Mur("source", "init", "mws", "--path", dir).OK(t)

	home.Mur("state", "push", "--source", "mws").OK(t)

	pushed := home.ReadFile("projects/mws/users/me@example.com/state.yaml")
	require.NotContains(t, pushed, "SECRET")
	require.Contains(t, pushed, "remote: https://oauth2@git.example.com/team/workspace.git")
}

func TestStatePush_ScanRootsTravelAsTildePaths(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("config", "set", "scan_roots", home.Path("code")+",/srv/repos").OK(t)

	home.Mur("state", "push", "--source", "mws").OK(t)

	require.Contains(t, home.ReadFile("projects/mws/users/me@example.com/state.yaml"), "scan_roots:\n  - ~/code\n  - /srv/repos\n")
}

func TestWorkspaceList_WithReposShowsEachWorkspacesRepos(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("workspace", "list", "--repos").OK(t)

	require.Contains(t, res.Stdout, "  repos:\n    - id: "+fixtures.CollectorID+"\n")
	require.Contains(t, res.Stdout, "    - id: "+fixtures.LogrotateID+"\n")
}

func TestRepoAdd_RewritesHandWrittenFlowListsInsideReposAsBlocks(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.WriteFile("projects/mws/workspaces/auditlogs/workspace.yaml", `summary: x
repos:
  - url: git@h:g/common.git
    description: Shared.
    aliases: [common]
`)

	home.Mur("repo", "add", "git@h:g/collector.git", "--workspace", "auditlogs", "--description", "Collector.").OK(t)

	require.Equal(t, `summary: x
repos:
  - url: git@h:g/common.git
    description: Shared.
    aliases:
      - common
  - url: git@h:g/collector.git
    description: Collector.
`, home.ReadFile("projects/mws/workspaces/auditlogs/workspace.yaml"))
}

func TestRepoRemove_TheLastRepoLeavesNoFlowEmptyList(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs", "--summary", "x").OK(t)
	home.Mur("repo", "add", "git@h:g/collector.git", "--workspace", "auditlogs", "--description", "Collector.").OK(t)

	home.Mur("repo", "remove", "collector", "--workspace", "auditlogs").OK(t)

	require.Equal(t, "summary: x\n", home.ReadFile("projects/mws/workspaces/auditlogs/workspace.yaml"))
}

func TestAMistypedSubcommandIsAUsageErrorNotASilentSuccess(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("workspace", "activat", "auditlogs")

	require.Equal(t, cli.ExitUsage, res.Code)
	require.Contains(t, res.Stderr, `unknown command "activat"`)
}

func TestSync_SaysWhenAScanRootDoesNotExist(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team(fixtures.TeamRepo{Name: "collector"})
	home.Mur("config", "set", "scan_roots", "~/projcets").OK(t)

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, "warning: scan root "+home.Path("projcets")+" does not exist")
}

func TestStatePull_AFailedSourceCloneNamesTheSourceAndItsRemote(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.WriteFile("projects/mws/users/me@example.com/state.yaml", `sources:
  - name: gone
    path: ~/sources/gone
    remote: `+home.Path("remotes/gone.git")+`
`)

	res := home.Mur("state", "pull", "--source", "mws")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "source gone: clone "+home.Path("remotes/gone.git"))
}

func TestSync_ARepoWhoseRemoteHasNoCommitsYetKeepsSyncing(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team()
	remote := home.EmptyRemote("fresh")
	home.Mur("repo", "add", "file://"+remote, "--workspace", "team", "--description", "Just created.").OK(t)
	home.Mur("repo", "locate", "fresh", "~/code/fresh", "--workspace", "team").OK(t)
	home.Mur("sync", "--workspace", "team").OK(t)

	res := home.Mur("sync", "--workspace", "team").OK(t)

	require.Contains(t, res.Stdout, "synced:\n  - id: "+fixtures.RepoID(remote)+"\n    branch: master\n")
	_, err := os.Stat(home.Path("code/fresh/.git"))
	require.NoError(t, err)
}
