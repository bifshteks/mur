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
	home.Auditlogs()
	home.Mur("workspace", "create", "modelhub").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/modelhub/gateway.git", "--workspace", "modelhub",
		"--description", "Model gateway.", "--alias", "gateway").OK(t)

	home.Mur("workspace", "activate", "modelhub").OK(t)
	home.Mur("resolve", "gateway").OK(t)
	require.Equal(t, cli.ExitNotFound, home.Mur("resolve", "collector").Code)

	home.Mur("workspace", "activate", "auditlogs").OK(t)
	home.Mur("resolve", "collector").OK(t)

	home.Setenv("MUR_WORKSPACE", "modelhub")
	home.Mur("resolve", "gateway").OK(t)
	home.Mur("resolve", "collector", "--workspace", "auditlogs").OK(t)

	list := home.Mur("workspace", "list").OK(t)
	require.YAMLEq(t, `
- name: auditlogs
  source: mws
  summary: Audit events of MWS Cloud.
  active: true
- name: modelhub
  source: mws
  summary: ""
`, list.Stdout)
}

func TestRead_WithNoWorkspaceAnywhereIsAUsageError(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("resolve", "collector")

	require.Equal(t, cli.ExitUsage, res.Code)
	require.Contains(t, res.Stderr, "--workspace")
}

func TestWorkspaceActivate_RefusesAWorkspaceThatDoesNotExist(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	require.Equal(t, cli.ExitNotFound, home.Mur("workspace", "activate", "billing").Code)
}

func TestWorkspace_SameNameInTwoSourcesMustBeQualified(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.Source("personal")
	home.Mur("workspace", "create", "auditlogs", "--source", "personal").OK(t)

	res := home.Mur("resolve", "collector", "--workspace", "auditlogs")

	require.Equal(t, cli.ExitNotFound, res.Code)
	require.YAMLEq(t, `
error: "workspace auditlogs is defined in several sources; use one of: mws/auditlogs, personal/auditlogs"
candidates:
  - mws/auditlogs
  - personal/auditlogs
`, res.Stderr)
	home.Mur("resolve", "collector", "--workspace", "mws/auditlogs").OK(t)
}
