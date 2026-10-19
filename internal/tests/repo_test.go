package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

const handWrittenWorkspace = `# Repos of the analytics team.
summary: Analytics events of Nimbus Cloud.
owner: analytics # not a key mur knows
statuses:
  - name: live
    description: Ships, deployed or used.
repos: []
`

func TestRepoAdd_KeepsCommentsAndUnknownKeysAndWritesBlockStyle(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.WriteFile("projects/nimbus/workspaces/analytics/workspace.yaml", handWrittenWorkspace)

	res := home.Mur("repo", "add", "git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git",
		"--workspace", "analytics",
		"--description", "Reads k8s node files and exports them.",
		"--alias", "collector",
		"--lang", "go",
		"--status", "live",
		"--notes", "Never log event bodies.",
	).OK(t)

	require.Contains(t, res.Stderr, `unknown key "owner"`)
	require.Equal(t, `# Repos of the analytics team.
summary: Analytics events of Nimbus Cloud.
owner: analytics # not a key mur knows
statuses:
  - name: live
    description: Ships, deployed or used.
repos:
  - url: git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git
    description: Reads k8s node files and exports them.
    aliases:
      - collector
    langs:
      - go
    status: live
    notes: Never log event bodies.
`, home.ReadFile("projects/nimbus/workspaces/analytics/workspace.yaml"))
}

func TestRepoAdd_RejectsAStatusTheWorkspaceDoesNotDeclare(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.WriteFile("projects/nimbus/workspaces/analytics/workspace.yaml", handWrittenWorkspace)

	res := home.Mur("repo", "add", "git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git",
		"--workspace", "analytics", "--description", "Collector.", "--status", "zombie")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "zombie")
}

func TestRepoAdd_RejectsAnAliasAnotherRepoInTheWorkspaceHas(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/services-team/svc-analytics-common.git",
		"--workspace", "analytics", "--description", "Shared code.", "--alias", "common").OK(t)

	res := home.Mur("repo", "add", "git@git.nimbus.example:nimbus/iam/common.git",
		"--workspace", "analytics", "--description", "IAM shared code.", "--alias", "common")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "common")
}

func TestRepoAdd_RejectsTheSameRepoTwiceEvenSpelledDifferently(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/services-team/svc-analytics-common.git",
		"--workspace", "analytics", "--description", "Shared code.").OK(t)

	res := home.Mur("repo", "add", "https://git.nimbus.example/nimbus/services-team/svc-analytics-common",
		"--workspace", "analytics", "--description", "Shared code.")

	require.Equal(t, cli.ExitError, res.Code)
}

func TestRepoAdd_IsAWriteSoItIgnoresTheEnvironmentWorkspace(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.Setenv("MUR_WORKSPACE", "analytics")

	res := home.Mur("repo", "add", "git@git.nimbus.example:nimbus/services-team/svc-analytics-common.git",
		"--description", "Shared code.")

	require.Equal(t, cli.ExitUsage, res.Code)
}

func TestWorkspaceCreate_RefusesANameThatExists(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)

	res := home.Mur("workspace", "create", "analytics")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "nimbus/analytics already exists")
}
