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
	home.Mur("source", "init", "nimbus", "--path", dir).OK(t)

	home.Mur("workspace", "create", "analytics", "--summary", "Analytics events of Nimbus Cloud.").OK(t)

	require.FileExists(t, filepath.Join(dir, "workspaces", "analytics", "workspace.yaml"))
	require.FileExists(t, filepath.Join(dir, "workspaces", "analytics", "README.md"))

	list := home.Mur("workspace", "list").OK(t)
	require.YAMLEq(t, `
- name: analytics
  source: nimbus
  summary: Analytics events of Nimbus Cloud.
`, list.Stdout)
}

func TestWorkspaceCreate_NeedsSourceFlagWhenSeveralSourcesAreAttached(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Mur("source", "init", "nimbus").OK(t)
	home.Mur("source", "init", "personal").OK(t)

	res := home.Mur("workspace", "create", "analytics")

	require.Equal(t, cli.ExitUsage, res.Code)
	require.Contains(t, res.Stderr, "--source")
}

func TestWorkspaceList_WithReposShowsEachWorkspacesRepos(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	res := home.Mur("workspace", "list", "--repos").OK(t)

	require.Contains(t, res.Stdout, "  repos:\n    - id: "+fixtures.CollectorID+"\n")
	require.Contains(t, res.Stdout, "    - id: "+fixtures.LogrotateID+"\n")
}
