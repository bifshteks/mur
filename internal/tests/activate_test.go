package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestWorkspaceActivate_ReadsFollowItUntilEnvOrFlagSayOtherwise(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Mur("workspace", "create", "modelhub").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/modelhub/gateway.git", "--workspace", "modelhub",
		"--description", "Model gateway.", "--alias", "gateway").OK(t)

	home.Mur("workspace", "activate", "modelhub").OK(t)
	home.Mur("resolve", "gateway").OK(t)
	require.Equal(t, cli.ExitNotFound, home.Mur("resolve", "collector").Code)

	home.Mur("workspace", "activate", "analytics").OK(t)
	home.Mur("resolve", "collector").OK(t)

	home.Setenv("MUR_WORKSPACE", "modelhub")
	home.Mur("resolve", "gateway").OK(t)
	home.Mur("resolve", "collector", "--workspace", "analytics").OK(t)

	list := home.Mur("workspace", "list").OK(t)
	require.YAMLEq(t, `
- name: analytics
  source: nimbus
  summary: Analytics events of Nimbus Cloud.
  active: true
- name: modelhub
  source: nimbus
  summary: ""
`, list.Stdout)
}

func TestRead_WithNoWorkspaceAnywhereIsAUsageError(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	res := home.Mur("resolve", "collector")

	require.Equal(t, cli.ExitUsage, res.Code)
	require.Contains(t, res.Stderr, "--workspace")
}

func TestWorkspaceActivate_RefusesAWorkspaceThatDoesNotExist(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	require.Equal(t, cli.ExitNotFound, home.Mur("workspace", "activate", "billing").Code)
}

func TestWorkspace_SameNameInTwoSourcesMustBeQualified(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Source("personal")
	home.Mur("workspace", "create", "analytics", "--source", "personal").OK(t)

	res := home.Mur("resolve", "collector", "--workspace", "analytics")

	require.Equal(t, cli.ExitNotFound, res.Code)
	require.YAMLEq(t, `
error: "workspace analytics is defined in several sources; use one of: nimbus/analytics, personal/analytics"
candidates:
  - nimbus/analytics
  - personal/analytics
`, res.Stderr)
	home.Mur("resolve", "collector", "--workspace", "nimbus/analytics").OK(t)
}
