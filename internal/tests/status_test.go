package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestStatusAdd_DeclaresStatusesAboveTheReposAndKeepsTheRestOfTheFile(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics", "--summary", "Analytics.").OK(t)
	home.Mur("repo", "add", fixtures.CollectorURL, "--workspace", "analytics", "--description", "Collector.").OK(t)

	res := home.Mur("status", "add", "live", "--workspace", "analytics", "--description", "Deployed.").OK(t)
	require.Equal(t, "name: live\ndescription: Deployed.\n", res.Stdout)

	res = home.Mur("status", "add", "dead", "--workspace", "analytics", "--description", "Superseded.", "--sync=false").OK(t)
	require.Equal(t, "name: dead\ndescription: Superseded.\nsync: false\n", res.Stdout)

	require.Equal(t, `summary: Analytics.
statuses:
  - name: live
    description: Deployed.
  - name: dead
    description: Superseded.
    sync: false
repos:
  - url: `+fixtures.CollectorURL+`
    description: Collector.
`, home.ReadFile("projects/nimbus/workspaces/analytics/workspace.yaml"))

	home.Mur("repo", "add", fixtures.LogrotateURL, "--workspace", "analytics",
		"--description", "Logrotate.", "--status", "dead").OK(t)
}

func TestStatusAdd_RejectsAStatusTheWorkspaceAlreadyDeclares(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.WriteFile("projects/nimbus/workspaces/analytics/workspace.yaml", handWrittenWorkspace)

	res := home.Mur("status", "add", "live", "--workspace", "analytics", "--description", "Again.")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "already declared")
}

func TestStatusAdd_IsAWriteSoItIgnoresTheEnvironmentWorkspace(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.Setenv("MUR_WORKSPACE", "analytics")

	res := home.Mur("status", "add", "live", "--description", "Deployed.")

	require.Equal(t, cli.ExitUsage, res.Code)
}
