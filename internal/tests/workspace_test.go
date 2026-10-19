package tests_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestWorkspaceCreate_ShowsUpInListWithItsSummary(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	dir := home.EmptyRepo("projects/workspace")
	home.Mur("source", "init", "mws", "--path", dir).OK(t)

	home.Mur("workspace", "create", "auditlogs", "--summary", "Audit events of MWS Cloud.").OK(t)

	require.FileExists(t, filepath.Join(dir, "workspaces", "auditlogs", "workspace.yaml"))
	require.FileExists(t, filepath.Join(dir, "workspaces", "auditlogs", "README.md"))

	list := home.Mur("workspace", "list").OK(t)
	require.YAMLEq(t, `
- name: auditlogs
  source: mws
  summary: Audit events of MWS Cloud.
`, list.Stdout)
}

func TestWorkspaceCreate_NeedsSourceFlagWhenSeveralSourcesAreAttached(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Mur("source", "init", "mws").OK(t)
	home.Mur("source", "init", "personal").OK(t)

	res := home.Mur("workspace", "create", "auditlogs")

	require.Equal(t, cli.ExitUsage, res.Code)
	require.Contains(t, res.Stderr, "--source")
}

func TestWorkspaceList_WithReposShowsEachWorkspacesRepos(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("workspace", "list", "--repos").OK(t)

	require.Contains(t, res.Stdout, "  repos:\n    - id: "+fixtures.CollectorID+"\n")
	require.Contains(t, res.Stdout, "    - id: "+fixtures.LogrotateID+"\n")
}
