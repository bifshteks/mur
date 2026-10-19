package tests_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestResolve_ByAliasAndBasenameGivesEverythingAboutTheRepos(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	collector := home.EmptyRepo("projects/services-team/svc-analytics-collector")
	home.Mur("repo", "locate", "collector", collector, "--workspace", "analytics").OK(t)
	home.Mur("repo", "locate", "svc-analytics-logrotate", "~/projects/services-team/svc-analytics-logrotate",
		"--workspace", "analytics").OK(t)

	res := home.Mur("resolve", "collector", "svc-analytics-logrotate", "--workspace", "analytics").OK(t)

	require.YAMLEq(t, `
- query: collector
  workspace: nimbus/analytics
  id: git.nimbus.example/nimbus/services-team/svc-analytics-collector
  url: git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git
  path: `+collector+`
  cloned: true
  description: Reads k8s node files and exports them.
  aliases:
    - collector
  langs:
    - go
  status: live
- query: svc-analytics-logrotate
  workspace: nimbus/analytics
  id: git.nimbus.example/nimbus/services-team/svc-analytics-logrotate
  url: git@git.nimbus.example:nimbus/services-team/svc-analytics-logrotate.git
  path: `+home.Path("projects/services-team/svc-analytics-logrotate")+`
  cloned: false
  description: Rotates analytics log files on nodes.
`, res.Stdout)
}

func TestResolve_ByAnySpellingOfTheURL(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	res := home.Mur("resolve", "https://git.nimbus.example/nimbus/services-team/svc-analytics-collector",
		"--workspace", "analytics", "--json").OK(t)

	var got []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal([]byte(res.Stdout), &got))
	require.Len(t, got, 1)
	require.Equal(t, fixtures.CollectorID, got[0].ID)
}

func TestResolve_PathPrintsOnlyAbsolutePathsForShellSubstitution(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Mur("repo", "locate", "collector", "/srv/collector", "--workspace", "analytics").OK(t)
	home.Mur("repo", "locate", "svc-analytics-logrotate", "/srv/logrotate", "--workspace", "analytics").OK(t)

	res := home.Mur("resolve", "collector", "svc-analytics-logrotate", "--workspace", "analytics", "--path").OK(t)

	require.Equal(t, "/srv/collector\n/srv/logrotate\n", res.Stdout)
}

func TestResolve_PathOfARepoWithNoRecordedPathNeedsInput(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	res := home.Mur("resolve", "collector", "--workspace", "analytics", "--path")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Empty(t, res.Stdout)
	require.YAMLEq(t, `
needs_path:
  - id: git.nimbus.example/nimbus/services-team/svc-analytics-collector
    alias: collector
    candidates: []
hint: mur repo locate <alias|url> <dir> --workspace analytics
`, res.Stderr)
}

func TestResolve_AnUnknownNameIsNotFoundAndPrintsNothingOnStdout(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()

	res := home.Mur("resolve", "collector", "exporter", "--workspace", "analytics")

	require.Equal(t, cli.ExitNotFound, res.Code)
	require.Empty(t, res.Stdout)
	require.YAMLEq(t, `
error: no repo named exporter in workspace nimbus/analytics
query: exporter
candidates: []
`, res.Stderr)
}

func TestResolve_ABasenameTwoReposShareIsAmbiguous(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "iam").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/iam/common.git", "--workspace", "iam", "--description", "IAM.").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/billing/common.git", "--workspace", "iam", "--description", "Billing.").OK(t)

	res := home.Mur("resolve", "common", "--workspace", "iam")

	require.Equal(t, cli.ExitNotFound, res.Code)
	require.YAMLEq(t, `
error: common matches several repos in workspace nimbus/iam
query: common
candidates:
  - git.nimbus.example/nimbus/iam/common
  - git.nimbus.example/nimbus/billing/common
`, res.Stderr)
}

func TestResolve_AnAliasWinsOverAnotherRepoBasename(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("nimbus")
	home.Mur("workspace", "create", "analytics").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/iam/common.git", "--workspace", "analytics",
		"--description", "IAM.").OK(t)
	home.Mur("repo", "add", "git@git.nimbus.example:nimbus/services-team/svc-analytics-common.git", "--workspace", "analytics",
		"--description", "Ours.", "--alias", "common").OK(t)
	home.Mur("repo", "locate", "common", "/srv/svc-analytics-common", "--workspace", "analytics").OK(t)

	res := home.Mur("resolve", "common", "--workspace", "analytics", "--path").OK(t)

	require.Equal(t, "/srv/svc-analytics-common\n", res.Stdout)
}
