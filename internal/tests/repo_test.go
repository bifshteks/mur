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
