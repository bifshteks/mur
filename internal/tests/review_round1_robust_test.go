package tests_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestSync_InterruptingACloneLeavesNoHalfClonedDirectory(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Team()
	url, stuck := home.HangingRemote()
	home.Mur("repo", "add", url, "--workspace", "team", "--description", "Behind a dead VPN.").OK(t)
	home.Mur("repo", "locate", "slow", "~/code/slow", "--workspace", "team").OK(t)

	ctx, interrupt := context.WithCancel(t.Context())
	done := make(chan fixtures.Result)

	go func() { done <- home.MurCtx(ctx, "sync", "--workspace", "team") }()

	<-stuck
	interrupt()

	select {
	case res := <-done:
		require.Equal(t, cli.ExitError, res.Code)
		require.Contains(t, res.Stdout, "context canceled")
	case <-time.After(20 * time.Second):
		t.Fatal("sync did not stop after the interrupt")
	}

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

	ctx, interrupt := context.WithCancel(t.Context())
	done := make(chan fixtures.Result)

	go func() { done <- home.MurCtx(ctx, "sync", "--workspace", "team") }()

	<-stuck
	home.Mur("config", "set", "scan_roots", "~/projects,~/work").OK(t)
	interrupt()
	<-done

	require.YAMLEq(t, "- ~/projects\n- ~/work\n", home.Mur("config", "get", "scan_roots").OK(t).Stdout)
	require.Equal(t, clone+"\n", home.Mur("resolve", "collector", "--workspace", "team", "--path").OK(t).Stdout)
}

func TestSync_AtATerminalAnAnswerMustBeThatRepoOrSomewhereEmpty(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	home.WriteFile("projects/services-team/README.md", "a parent directory, not a clone\n")
	home.Mkdir("code/gateway")

	// collector: a parent directory is refused, then skipped;
	// gateway: an existing empty directory is cloned into.
	res := home.MurTTY("~/projects/services-team\n\n~/code/gateway\n", "sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Contains(t, res.Stderr, home.Path("projects/services-team")+" is not a clone of "+fixtures.RepoID(remotes["collector"]))
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
