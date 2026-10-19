package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

const handWrittenWorkspace = `# Repos of the auditlogs team.
summary: Audit events of MWS Cloud.
owner: auditlogs # not a key mur knows
statuses:
  - name: live
    description: Ships, deployed or used.
repos: []
`

func TestRepoAdd_KeepsCommentsAndUnknownKeysAndWritesBlockStyle(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.WriteFile("projects/mws/workspaces/auditlogs/workspace.yaml", handWrittenWorkspace)

	res := home.Mur("repo", "add", "git@git.mws-team.ru:mws/security-services/sec-audit-collector.git",
		"--workspace", "auditlogs",
		"--description", "Reads k8s node files and exports them.",
		"--alias", "collector",
		"--lang", "go",
		"--status", "live",
		"--notes", "Never log event bodies.",
	).OK(t)

	require.Contains(t, res.Stderr, `unknown key "owner"`)
	require.Equal(t, `# Repos of the auditlogs team.
summary: Audit events of MWS Cloud.
owner: auditlogs # not a key mur knows
statuses:
  - name: live
    description: Ships, deployed or used.
repos:
  - url: git@git.mws-team.ru:mws/security-services/sec-audit-collector.git
    description: Reads k8s node files and exports them.
    aliases:
      - collector
    langs:
      - go
    status: live
    notes: Never log event bodies.
`, home.ReadFile("projects/mws/workspaces/auditlogs/workspace.yaml"))
}

func TestRepoAdd_RejectsAStatusTheWorkspaceDoesNotDeclare(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.WriteFile("projects/mws/workspaces/auditlogs/workspace.yaml", handWrittenWorkspace)

	res := home.Mur("repo", "add", "git@git.mws-team.ru:mws/security-services/sec-audit-collector.git",
		"--workspace", "auditlogs", "--description", "Collector.", "--status", "zombie")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "zombie")
}

func TestRepoAdd_RejectsAnAliasAnotherRepoInTheWorkspaceHas(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/security-services/sec-audit-common.git",
		"--workspace", "auditlogs", "--description", "Shared code.", "--alias", "common").OK(t)

	res := home.Mur("repo", "add", "git@git.mws-team.ru:mws/iam/common.git",
		"--workspace", "auditlogs", "--description", "IAM shared code.", "--alias", "common")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "common")
}

func TestRepoAdd_RejectsTheSameRepoTwiceEvenSpelledDifferently(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/security-services/sec-audit-common.git",
		"--workspace", "auditlogs", "--description", "Shared code.").OK(t)

	res := home.Mur("repo", "add", "https://git.mws-team.ru/mws/security-services/sec-audit-common",
		"--workspace", "auditlogs", "--description", "Shared code.")

	require.Equal(t, cli.ExitError, res.Code)
}

func TestRepoAdd_IsAWriteSoItIgnoresTheEnvironmentWorkspace(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.Setenv("MUR_WORKSPACE", "auditlogs")

	res := home.Mur("repo", "add", "git@git.mws-team.ru:mws/security-services/sec-audit-common.git",
		"--description", "Shared code.")

	require.Equal(t, cli.ExitUsage, res.Code)
}

func TestWorkspaceCreate_RefusesANameThatExists(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)

	res := home.Mur("workspace", "create", "auditlogs")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "mws/auditlogs already exists")
}

func TestRepoRemove_DropsOnlyThatEntryAndKeepsTheRestOfTheFile(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	home.Mur("repo", "remove", "collector", "--workspace", "auditlogs").OK(t)

	require.Equal(t, `summary: Audit events of MWS Cloud.
statuses:
  - name: live
    description: Ships, deployed or used.
  - name: dead
    description: Superseded. Do not change.
    sync: false
repos:
  - url: git@git.mws-team.ru:mws/security-services/sec-audit-logrotate.git
    description: Rotates audit log files on nodes.
`, home.ReadFile("projects/mws/workspaces/auditlogs/workspace.yaml"))
	require.Equal(t, cli.ExitNotFound, home.Mur("resolve", "collector", "--workspace", "auditlogs").Code)
}

func TestRepoRemove_IsAWriteAndNeedsTheWorkspaceFlag(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.Mur("workspace", "activate", "auditlogs").OK(t)

	require.Equal(t, cli.ExitUsage, home.Mur("repo", "remove", "collector").Code)
}

func TestRepoList_ShowsEveryRepoOfTheWorkspace(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("repo", "list", "--workspace", "auditlogs").OK(t)

	require.YAMLEq(t, `
- id: git.mws-team.ru/mws/security-services/sec-audit-collector
  url: git@git.mws-team.ru:mws/security-services/sec-audit-collector.git
  cloned: false
  description: Reads k8s node files and exports them.
  aliases:
    - collector
  langs:
    - go
  status: live
- id: git.mws-team.ru/mws/security-services/sec-audit-logrotate
  url: git@git.mws-team.ru:mws/security-services/sec-audit-logrotate.git
  cloned: false
  description: Rotates audit log files on nodes.
`, res.Stdout)
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

func TestRepoLocate_RefusesAnotherReposClone(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	clone := home.Clone(remotes["collector"], "projects/collector")

	res := home.Mur("repo", "locate", "gateway", clone, "--workspace", "team")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, clone+" is not a clone of "+fixtures.RepoID(remotes["gateway"]))
	require.Equal(t, cli.ExitNeedsPath, home.Mur("resolve", "gateway", "--workspace", "team", "--path").Code)
}

func TestRepoLocate_RefusesAPathAnotherRepoAlreadyHas(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"})
	home.Mur("repo", "locate", "collector", "~/code/svc", "--workspace", "team").OK(t)

	res := home.Mur("repo", "locate", "gateway", "~/code/svc", "--workspace", "team")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, home.Path("code/svc")+" is already the path of "+fixtures.RepoID(remotes["collector"]))
}

func TestRepoLocate_AcceptsItsOwnCloneAMissingOrAnEmptyDirectory(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	remotes := home.Team(fixtures.TeamRepo{Name: "collector"}, fixtures.TeamRepo{Name: "gateway"}, fixtures.TeamRepo{Name: "billing"})
	clone := home.Clone(remotes["collector"], "projects/collector")
	home.Mkdir("code/gateway")

	require.Contains(t, home.Mur("repo", "locate", "collector", clone, "--workspace", "team").OK(t).Stdout, "cloned: true")
	require.Contains(t, home.Mur("repo", "locate", "gateway", "~/code/gateway", "--workspace", "team").OK(t).Stdout, "cloned: false")
	require.Contains(t, home.Mur("repo", "locate", "billing", "~/code/billing", "--workspace", "team").OK(t).Stdout, "cloned: false")
}
