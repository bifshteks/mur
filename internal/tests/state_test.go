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
sources:
  - name: nimbus
    path: ~/projects/nimbus
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
