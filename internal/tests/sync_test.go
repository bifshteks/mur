package tests_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestSync_AdoptsTheOneExistingCloneWhoseOriginMatches(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"})
	clone := home.Clone(remotes["collector"], "projects/security-services/collector")

	res := home.Mur("sync", "--workspace", "team").OK(t)

	require.YAMLEq(t, `
synced: []
cloned: []
adopted:
  - id: `+fixtures.RepoID(remotes["collector"])+`
    path: `+clone+`
skipped: []
failed: []
needs_path: []
`, res.Stdout)
	require.Equal(t, clone+"\n", home.Mur("resolve", "collector", "--workspace", "team", "--path").OK(t).Stdout)
}

func TestSync_ClonesALocatedRepoThatIsNotThereYet(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team(fixtures.TeamRepo{Name: "logrotate"})
	home.Mur("repo", "locate", "logrotate", "~/code/logrotate", "--workspace", "team").OK(t)

	res := home.Mur("sync", "--workspace", "team").OK(t)

	require.Contains(t, res.Stdout, "cloned:\n  - id: ")
	require.Contains(t, res.Stdout, "path: "+home.Path("code/logrotate"))
	require.Equal(t, "initial commit", home.Git(home.Path("code/logrotate"), "log", "-1", "--format=%s"))
}

func TestSync_FetchesExistingClonesWithoutTouchingTheWorkingTree(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"})
	clone := home.Clone(remotes["collector"], "projects/collector")
	home.Mur("repo", "locate", "collector", clone, "--workspace", "team").OK(t)
	home.PushCommit(remotes["collector"], "teammate change")

	res := home.Mur("sync", "--workspace", "team").OK(t)

	require.YAMLEq(t, `
synced:
  - id: `+fixtures.RepoID(remotes["collector"])+`
    branch: master
    dirty: false
    ahead: 0
    behind: 1
cloned: []
adopted: []
skipped: []
failed: []
needs_path: []
`, res.Stdout)
	require.Equal(t, "initial commit", home.Git(clone, "log", "-1", "--format=%s"))
}

func TestSync_WithoutATerminalSyncsTheRestThenAsksForWhatItCannotPlace(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	home.Clone(remotes["collector"], "projects/collector")

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stdout, "adopted:\n  - id: ")
	require.Contains(t, res.Stdout, `needs_path:
  - id: `+fixtures.RepoID(remotes["gateway"])+`
    alias: gateway
    candidates: []
hint: mur repo locate <alias|url> <dir> --workspace team
`)
}

func TestSync_TwoClonesOfOneRepoAreCandidatesNotAGuess(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"})
	first := home.Clone(remotes["collector"], "projects/a/collector")
	second := home.Clone(remotes["collector"], "projects/b/collector")

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stdout, "candidates:\n      - "+first+"\n      - "+second+"\n")
}

func TestSync_SkipsStatusesThatOptOutUnlessAskedForAll(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector", Status: "live"}, fixtures.TeamRepo{Name: "exporter", Status: "dead"})
	home.Clone(remotes["collector"], "projects/collector")
	home.Clone(remotes["exporter"], "projects/exporter")
	exporterID := fixtures.RepoID(remotes["exporter"])

	res := home.Mur("sync", "--workspace", "team").OK(t)
	require.Contains(t, res.Stdout, "skipped:\n  - id: "+exporterID+"\n    reason: status dead has sync false\n")

	res = home.Mur("sync", "--workspace", "team", "--status", "dead").OK(t)
	require.Contains(t, res.Stdout, "adopted:\n  - id: "+exporterID+"\n")
	require.NotContains(t, res.Stdout, "collector")

	res = home.Mur("sync", "--workspace", "team", "--all").OK(t)
	require.Contains(t, res.Stdout, "synced:\n  - id: ")
	require.Contains(t, res.Stdout, "skipped: []")
}

func TestSync_AWorktreeIsNotASecondCandidate(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"})
	clone := home.Clone(remotes["collector"], "projects/collector")
	home.Git(clone, "worktree", "add", "-q", "-b", "feature", home.Path("projects/wt/collector-feature"))

	res := home.Mur("sync", "--workspace", "team").OK(t)

	require.Contains(t, res.Stdout, "path: "+clone+"\n")
}

func TestSync_OneBrokenRepoFailsAloneAndTheExitCodeSaysSo(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team(fixtures.TeamRepo{Name: "collector"})
	home.Mur("repo", "add", "file://"+home.Path("remotes/gone.git"), "--workspace", "team", "--description", "Deleted.").OK(t)
	home.Mur("repo", "locate", "gone", "~/code/gone", "--workspace", "team").OK(t)
	home.Mur("repo", "locate", "collector", "~/code/collector", "--workspace", "team").OK(t)

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stdout, "cloned:\n  - id: ")
	require.Contains(t, res.Stdout, "failed:\n  - id: "+home.Path("remotes/gone")+"\n    error: ")
}

func TestSync_AtATerminalAsksForEachRepoItCannotPlace(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"}, fixtures.TeamRepo{Name: "billing"})
	home.Clone(remotes["collector"], "projects/a/collector")
	second := home.Clone(remotes["collector"], "projects/b/collector")

	// collector: pick the second candidate; gateway: type a new path; billing: skip.
	res := home.MurTTY("2\n~/code/gateway\n\n", "sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, "collector")
	require.Contains(t, res.Stdout, "adopted:\n  - id: "+fixtures.RepoID(remotes["collector"])+"\n    path: "+second+"\n")
	require.Contains(t, res.Stdout, "cloned:\n  - id: "+fixtures.RepoID(remotes["gateway"])+"\n")
	require.Contains(t, res.Stdout, "needs_path:\n  - id: "+fixtures.RepoID(remotes["billing"])+"\n")
	require.Equal(t, "initial commit", home.Git(home.Path("code/gateway"), "log", "-1", "--format=%s"))
}

func TestSyncPull_FastForwardsOnlyACleanDefaultBranch(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "clean"}, fixtures.TeamRepo{Name: "dirty"}, fixtures.TeamRepo{Name: "busy"})
	clean := home.Clone(remotes["clean"], "projects/clean")
	dirty := home.Clone(remotes["dirty"], "projects/dirty")
	busy := home.Clone(remotes["busy"], "projects/busy")
	home.WriteFile("projects/dirty/wip.txt", "half done\n")
	home.Git(busy, "checkout", "-q", "-b", "feature")
	home.Mur("sync", "--workspace", "team").OK(t)

	for _, r := range remotes {
		home.PushCommit(r, "teammate change")
	}

	res := home.Mur("sync", "--workspace", "team", "--pull").OK(t)

	require.Equal(t, "teammate change", home.Git(clean, "log", "-1", "--format=%s"))
	require.Equal(t, "initial commit", home.Git(dirty, "log", "-1", "--format=%s"))
	require.Equal(t, "initial commit", home.Git(busy, "log", "-1", "--format=%s"))
	require.Contains(t, res.Stdout, "reason: 'fetched, not pulled: working tree dirty'")
	require.Contains(t, res.Stdout, "reason: 'fetched, not pulled: on feature, default is master'")
}

func TestSyncDryRun_ChangesNothing(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	home.Clone(remotes["collector"], "projects/collector")
	home.Mur("repo", "locate", "gateway", "~/code/gateway", "--workspace", "team").OK(t)
	before := home.Mur("resolve", "collector", "gateway", "--workspace", "team").OK(t).Stdout

	res := home.Mur("sync", "--workspace", "team", "--dry-run").OK(t)

	require.Contains(t, res.Stdout, "adopted:\n  - id: ")
	require.Contains(t, res.Stdout, "cloned:\n  - id: ")
	require.NoDirExists(t, home.Path("code/gateway"))
	require.Equal(t, before, home.Mur("resolve", "collector", "gateway", "--workspace", "team").OK(t).Stdout)
}

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

func TestSync_SaysWhenAScanRootDoesNotExist(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team(fixtures.TeamRepo{Name: "collector"})
	home.Mur("config", "set", "scan_roots", "~/projcets").OK(t)

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, "warning: scan root "+home.Path("projcets")+" does not exist")
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

func TestSync_InterruptingACloneLeavesNoHalfClonedDirectory(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team()
	url, stuck := home.HangingRemote()
	home.Mur("repo", "add", url, "--workspace", "team", "--description", "Behind a dead VPN.").OK(t)
	home.Mur("repo", "locate", "slow", "~/code/slow", "--workspace", "team").OK(t)

	res := home.MurInterrupted(stuck, nil, "sync", "--workspace", "team")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stdout, "context canceled")
	require.NoDirExists(t, home.Path("code/slow"))
}

func TestSync_KeepsAStateChangeAnotherMurMadeWhileItRan(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"})
	clone := home.Clone(remotes["collector"], "projects/collector")
	url, stuck := home.HangingRemote()
	home.Mur("repo", "add", url, "--workspace", "team", "--description", "Behind a dead VPN.").OK(t)
	home.Mur("repo", "locate", "slow", "~/code/slow", "--workspace", "team").OK(t)

	home.MurInterrupted(stuck, func() {
		home.Mur("config", "set", "scan_roots", "~/projects,~/work").OK(t)
	}, "sync", "--workspace", "team")

	require.YAMLEq(t, "- ~/projects\n- ~/work\n", home.Mur("config", "get", "scan_roots").OK(t).Stdout)
	require.Equal(t, clone+"\n", home.Mur("resolve", "collector", "--workspace", "team", "--path").OK(t).Stdout)
}

func TestSync_AtATerminalAnAnswerMustBeThatRepoOrSomewhereEmpty(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	home.WriteFile("projects/security-services/README.md", "a parent directory, not a clone\n")
	home.Mkdir("code/gateway")

	// collector: a parent directory is refused, then skipped;
	// gateway: an existing empty directory is cloned into.
	res := home.MurTTY("~/projects/security-services\n\n~/code/gateway\n", "sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, home.Path("projects/security-services")+" is not a clone of "+fixtures.RepoID(remotes["collector"]))
	require.Contains(t, res.Stdout, "needs_path:\n  - id: "+fixtures.RepoID(remotes["collector"])+"\n")
	require.Contains(t, res.Stdout, "cloned:\n  - id: "+fixtures.RepoID(remotes["gateway"])+"\n")
	require.Equal(t, "initial commit", home.Git(home.Path("code/gateway"), "log", "-1", "--format=%s"))
}

func TestSync_AnInterruptAtThePromptStopsTheQuestions(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})

	ctx, interrupt := context.WithCancel(t.Context())
	interrupt()

	res := home.MurTTYCtx(ctx, "", "sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.NotContains(t, res.Stderr, "has no path")
	require.Contains(t, res.Stdout, "alias: collector")
	require.Contains(t, res.Stdout, "alias: gateway")
}

func TestSync_APathHoldingADifferentRepoFailsInsteadOfPassingForIt(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	clone := home.Clone(remotes["collector"], "projects/collector")
	home.Mur("repo", "locate", "collector", clone, "--workspace", "team").OK(t)
	home.Mur("repo", "locate", "gateway", clone, "--workspace", "team").OK(t)

	res := home.Mur("sync", "--workspace", "team")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stdout, "synced:\n  - id: "+fixtures.RepoID(remotes["collector"])+"\n")
	require.Contains(t, res.Stdout, "failed:\n  - id: "+fixtures.RepoID(remotes["gateway"])+"\n    error: "+
		clone+" holds "+fixtures.RepoID(remotes["collector"]))
}

func TestSync_AtATerminalOnePathCannotBeGivenToTwoRepos(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})

	res := home.MurTTY("~/code/svc\n~/code/svc\n\n", "sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, home.Path("code/svc")+" is already the path of "+fixtures.RepoID(remotes["collector"]))
	require.Contains(t, res.Stdout, "cloned:\n  - id: "+fixtures.RepoID(remotes["collector"])+"\n")
	require.Contains(t, res.Stdout, "needs_path:\n  - id: "+fixtures.RepoID(remotes["gateway"])+"\n")
}

func TestSync_AtATerminalANumberMustBeOneOfTheCandidates(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team(fixtures.TeamRepo{Name: "collector"})

	res := home.MurTTY("3\n\n", "sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, "3 is not one of the 0 candidates")
	require.Equal(t, cli.ExitNeedsPath, home.Mur("resolve", "collector", "--workspace", "team", "--path").Code)
}
