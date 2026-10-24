package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestStatePush_WritesThisLaptopUnderTheGitEmailWithTildePaths(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Mur("workspace", "activate", "analytics").OK(t)
	home.Mur("repo", "locate", "collector", "~/projects/services-team/svc-analytics-collector").OK(t)

	home.Mur("state", "push", "--source", "nimbus").OK(t)

	require.Equal(t, `active: analytics
paths:
  git.nimbus.example/nimbus/services-team/svc-analytics-collector: ~/projects/services-team/svc-analytics-collector
`, home.ReadFile("projects/nimbus/users/me@example.com/state.yaml"))
}

func TestStatePush_UserFlagChoosesTheKey(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	home.Mur("state", "push", "--source", "nimbus", "--user", "laptop-2@example.com").OK(t)

	require.FileExists(t, home.Path("projects/nimbus/users/laptop-2@example.com/state.yaml"))
}

func TestNewLaptop_SourceAddStatePullSyncRestoresTheWorkspace(t *testing.T) {
	t.Parallel()

	old := fixtures.NewHome(t)
	remotes := old.Team(
		fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"}, fixtures.TeamRepo{Name: "billing"},
	)
	old.Clone(remotes["collector"], "projects/collector")
	old.Clone(remotes["gateway"], "projects/platform/gateway")
	old.Mur("workspace", "activate", "team").OK(t)
	require.Equal(t, cli.ExitNeedsPath, old.Mur("sync").Code, "billing was never cloned on the old laptop")
	old.Mur("state", "push", "--source", "nimbus").OK(t)
	url := old.PublishSource(old.Path("projects/nimbus"), "nimbus")

	fresh := fixtures.NewHome(t)
	fresh.Mur("source", "add", "nimbus", url).OK(t)
	fresh.Mur("state", "pull", "--source", "nimbus").OK(t)
	res := fresh.Mur("sync")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Equal(t, "initial commit", fresh.Git(fresh.Path("projects/collector"), "log", "-1", "--format=%s"))
	require.Equal(t, "initial commit", fresh.Git(fresh.Path("projects/platform/gateway"), "log", "-1", "--format=%s"))
	require.Contains(t, res.Stdout, "needs_path:\n  - id: "+fixtures.RepoID(remotes["billing"])+"\n")
}

func TestStatePull_KeepsWhatThisLaptopAlreadyDecided(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Mur("workspace", "create", "modelhub").OK(t)
	home.WriteFile("projects/nimbus/users/me@example.com/state.yaml", `active: modelhub
sources:
  - name: nimbus
    path: ~/elsewhere/nimbus
paths:
  git.nimbus.example/nimbus/services-team/svc-analytics-collector: ~/old/collector
  git.nimbus.example/nimbus/services-team/svc-analytics-logrotate: ~/old/logrotate
`)
	home.Mur("workspace", "activate", "analytics").OK(t)
	home.Mur("repo", "locate", "collector", "~/mine/collector").OK(t)

	home.Mur("state", "pull", "--source", "nimbus").OK(t)

	require.Equal(t, home.Path("mine/collector")+"\n"+home.Path("old/logrotate")+"\n",
		home.Mur("resolve", "collector", "svc-analytics-logrotate", "--path").OK(t).Stdout)
	require.Contains(t, home.Mur("workspace", "list").OK(t).Stdout, "name: analytics\n  source: nimbus\n  summary: Analytics events of Nimbus Cloud.\n  active: true")
	require.YAMLEq(t, "- name: nimbus\n  path: "+home.Path("projects/nimbus")+"\n", home.Mur("source", "list").OK(t).Stdout)
}

func TestConfigScanRoots_DecideWhereSyncLooksForClones(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"})
	clone := home.Clone(remotes["collector"], "code/collector")
	require.YAMLEq(t, "- ~/projects\n", home.Mur("config", "get", "scan_roots").OK(t).Stdout)

	home.Mur("config", "set", "scan_roots", "~/code,~/work").OK(t)

	require.YAMLEq(t, "- ~/code\n- ~/work\n", home.Mur("config", "get", "scan_roots").OK(t).Stdout)
	require.Contains(t, home.Mur("sync", "--workspace", "team").OK(t).Stdout, "path: "+clone+"\n")
}

func TestStatePush_NeverPublishesTheSourcesList(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	dir := home.EmptyRepo("projects/mine")
	home.Git(dir, "remote", "add", "origin", "https://oauth2:glpat-SECRET@git.example.com/me/workspace.git")
	home.Mur("source", "init", "mine", "--path", dir).OK(t)

	home.Mur("state", "push", "--source", "nimbus").OK(t)

	pushed := home.ReadFile("projects/nimbus/users/me@example.com/state.yaml")
	require.NotContains(t, pushed, "sources:")
	require.NotContains(t, pushed, "mine")
	require.NotContains(t, pushed, "SECRET")
}

func TestStatePush_CarriesOnlyWhatTheSourceDefines(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Source("mine")
	home.Mur("workspace", "create", "hobby", "--source", "mine").OK(t)
	home.Mur("repo", "add", "git@github.com:me/blog.git", "--workspace", "hobby", "--description", "My blog.").OK(t)
	home.Mur("repo", "locate", "collector", "~/work/collector", "--workspace", "analytics").OK(t)
	home.Mur("repo", "locate", "blog", "~/fun/blog", "--workspace", "hobby").OK(t)
	home.Mur("workspace", "activate", "hobby").OK(t)

	home.Mur("state", "push", "--source", "nimbus").OK(t)
	home.Mur("state", "push", "--source", "mine").OK(t)

	require.Equal(t, `paths:
  git.nimbus.example/nimbus/services-team/svc-analytics-collector: ~/work/collector
`, home.ReadFile("projects/nimbus/users/me@example.com/state.yaml"))
	require.Equal(t, `active: hobby
paths:
  github.com/me/blog: ~/fun/blog
`, home.ReadFile("projects/mine/users/me@example.com/state.yaml"))
}

func TestStatePull_WithoutSourceMergesEveryAttachedSource(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Source("mine")
	home.Mur("workspace", "create", "hobby", "--source", "mine").OK(t)
	home.Mur("repo", "add", "git@github.com:me/blog.git", "--workspace", "hobby", "--description", "My blog.").OK(t)
	home.WriteFile("projects/nimbus/users/me@example.com/state.yaml", `paths:
  git.nimbus.example/nimbus/services-team/svc-analytics-collector: ~/work/collector
`)
	home.WriteFile("projects/mine/users/me@example.com/state.yaml", `active: hobby
paths:
  github.com/me/blog: ~/fun/blog
`)

	res := home.Mur("state", "pull").OK(t)

	require.Contains(t, res.Stdout, "active: hobby\n")
	require.Equal(t, home.Path("work/collector")+"\n",
		home.Mur("resolve", "collector", "--workspace", "analytics", "--path").OK(t).Stdout)
	require.Equal(t, home.Path("fun/blog")+"\n", home.Mur("resolve", "blog", "--path").OK(t).Stdout)
}

func TestStatePull_WithoutSourceSkipsSourcesWithNoSharedState(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Source("mine")

	require.Equal(t, cli.ExitNotFound, home.Mur("state", "pull").Code)

	home.WriteFile("projects/mine/users/me@example.com/state.yaml", "scan_roots:\n  - ~/code\n")

	require.Contains(t, home.Mur("state", "pull").OK(t).Stdout, "users/me@example.com/state.yaml\n")
}

func TestStatePush_ScanRootsTravelAsTildePaths(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("config", "set", "scan_roots", home.Path("code")+",/srv/repos").OK(t)

	home.Mur("state", "push", "--source", "nimbus").OK(t)

	require.Contains(t, home.ReadFile("projects/nimbus/users/me@example.com/state.yaml"), "scan_roots:\n  - ~/code\n  - /srv/repos\n")
}

func TestStatePull_AFailedSourceCloneNamesTheSourceAndItsRemote(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.WriteFile("projects/nimbus/users/me@example.com/state.yaml", `sources:
  - name: gone
    path: ~/sources/gone
    remote: `+home.Path("remotes/gone.git")+`
`)

	res := home.Mur("state", "pull", "--source", "nimbus")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "source gone: clone "+home.Path("remotes/gone.git"))
}
