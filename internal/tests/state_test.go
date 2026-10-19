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
	home.Auditlogs()
	home.Mur("workspace", "activate", "auditlogs").OK(t)
	home.Mur("repo", "locate", "collector", "~/projects/security-services/sec-audit-collector").OK(t)

	home.Mur("state", "push", "--source", "mws").OK(t)

	require.Equal(t, `active: auditlogs
sources:
  - name: mws
    path: ~/projects/mws
paths:
  git.mws-team.ru/mws/security-services/sec-audit-collector: ~/projects/security-services/sec-audit-collector
`, home.ReadFile("projects/mws/users/me@example.com/state.yaml"))
}

func TestStatePush_UserFlagChoosesTheKey(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	home.Mur("state", "push", "--source", "mws", "--user", "laptop-2@example.com").OK(t)

	require.FileExists(t, home.Path("projects/mws/users/laptop-2@example.com/state.yaml"))
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
	old.Mur("state", "push", "--source", "mws").OK(t)
	url := old.PublishSource(old.Path("projects/mws"), "mws")

	fresh := fixtures.NewHome(t)
	fresh.Mur("source", "add", "mws", url).OK(t)
	fresh.Mur("state", "pull", "--source", "mws").OK(t)
	res := fresh.Mur("sync")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Equal(t, "initial commit", fresh.Git(fresh.Path("projects/collector"), "log", "-1", "--format=%s"))
	require.Equal(t, "initial commit", fresh.Git(fresh.Path("projects/platform/gateway"), "log", "-1", "--format=%s"))
	require.Contains(t, res.Stdout, "needs_path:\n  - id: "+fixtures.RepoID(remotes["billing"])+"\n")
}

func TestStatePull_KeepsWhatThisLaptopAlreadyDecided(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.Mur("workspace", "create", "modelhub").OK(t)
	home.WriteFile("projects/mws/users/me@example.com/state.yaml", `active: modelhub
sources:
  - name: mws
    path: ~/elsewhere/mws
paths:
  git.mws-team.ru/mws/security-services/sec-audit-collector: ~/old/collector
  git.mws-team.ru/mws/security-services/sec-audit-logrotate: ~/old/logrotate
`)
	home.Mur("workspace", "activate", "auditlogs").OK(t)
	home.Mur("repo", "locate", "collector", "~/mine/collector").OK(t)

	home.Mur("state", "pull", "--source", "mws").OK(t)

	require.Equal(t, home.Path("mine/collector")+"\n"+home.Path("old/logrotate")+"\n",
		home.Mur("resolve", "collector", "sec-audit-logrotate", "--path").OK(t).Stdout)
	require.Contains(t, home.Mur("workspace", "list").OK(t).Stdout, "name: auditlogs\n  source: mws\n  summary: Audit events of MWS Cloud.\n  active: true")
	require.YAMLEq(t, "- name: mws\n  path: "+home.Path("projects/mws")+"\n", home.Mur("source", "list").OK(t).Stdout)
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
