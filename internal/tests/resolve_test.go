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
	home.Auditlogs()
	collector := home.EmptyRepo("projects/security-services/sec-audit-collector")
	home.Mur("repo", "locate", "collector", collector, "--workspace", "auditlogs").OK(t)
	home.Mur("repo", "locate", "sec-audit-logrotate", "~/projects/security-services/sec-audit-logrotate",
		"--workspace", "auditlogs").OK(t)

	res := home.Mur("resolve", "collector", "sec-audit-logrotate", "--workspace", "auditlogs").OK(t)

	require.YAMLEq(t, `
- query: collector
  workspace: mws/auditlogs
  id: git.mws-team.ru/mws/security-services/sec-audit-collector
  url: git@git.mws-team.ru:mws/security-services/sec-audit-collector.git
  path: `+collector+`
  cloned: true
  description: Reads k8s node files and exports them.
  aliases:
    - collector
  langs:
    - go
  status: live
- query: sec-audit-logrotate
  workspace: mws/auditlogs
  id: git.mws-team.ru/mws/security-services/sec-audit-logrotate
  url: git@git.mws-team.ru:mws/security-services/sec-audit-logrotate.git
  path: `+home.Path("projects/security-services/sec-audit-logrotate")+`
  cloned: false
  description: Rotates audit log files on nodes.
`, res.Stdout)
}

func TestResolve_ByAnySpellingOfTheURL(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("resolve", "https://git.mws-team.ru/mws/security-services/sec-audit-collector",
		"--workspace", "auditlogs", "--json").OK(t)

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
	home.Auditlogs()
	home.Mur("repo", "locate", "collector", "/srv/collector", "--workspace", "auditlogs").OK(t)
	home.Mur("repo", "locate", "sec-audit-logrotate", "/srv/logrotate", "--workspace", "auditlogs").OK(t)

	res := home.Mur("resolve", "collector", "sec-audit-logrotate", "--workspace", "auditlogs", "--path").OK(t)

	require.Equal(t, "/srv/collector\n/srv/logrotate\n", res.Stdout)
}

func TestResolve_PathOfARepoWithNoRecordedPathNeedsInput(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("resolve", "collector", "--workspace", "auditlogs", "--path")

	require.Equal(t, cli.ExitNeedsPath, res.Code)
	require.Empty(t, res.Stdout)
	require.YAMLEq(t, `
needs_path:
  - id: git.mws-team.ru/mws/security-services/sec-audit-collector
    alias: collector
    candidates: []
hint: mur repo locate <alias|url> <dir> --workspace auditlogs
`, res.Stderr)
}

func TestResolve_AnUnknownNameIsNotFoundAndPrintsNothingOnStdout(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("resolve", "collector", "exporter", "--workspace", "auditlogs")

	require.Equal(t, cli.ExitNotFound, res.Code)
	require.Empty(t, res.Stdout)
	require.YAMLEq(t, `
error: no repo named exporter in workspace mws/auditlogs
query: exporter
candidates: []
`, res.Stderr)
}

func TestResolve_ABasenameTwoReposShareIsAmbiguous(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "iam").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/iam/common.git", "--workspace", "iam", "--description", "IAM.").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/billing/common.git", "--workspace", "iam", "--description", "Billing.").OK(t)

	res := home.Mur("resolve", "common", "--workspace", "iam")

	require.Equal(t, cli.ExitNotFound, res.Code)
	require.YAMLEq(t, `
error: common matches several repos in workspace mws/iam
query: common
candidates:
  - git.mws-team.ru/mws/iam/common
  - git.mws-team.ru/mws/billing/common
`, res.Stderr)
}

func TestResolve_AnAliasWinsOverAnotherRepoBasename(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Source("mws")
	home.Mur("workspace", "create", "auditlogs").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/iam/common.git", "--workspace", "auditlogs",
		"--description", "IAM.").OK(t)
	home.Mur("repo", "add", "git@git.mws-team.ru:mws/security-services/sec-audit-common.git", "--workspace", "auditlogs",
		"--description", "Ours.", "--alias", "common").OK(t)
	home.Mur("repo", "locate", "common", "/srv/sec-audit-common", "--workspace", "auditlogs").OK(t)

	res := home.Mur("resolve", "common", "--workspace", "auditlogs", "--path").OK(t)

	require.Equal(t, "/srv/sec-audit-common\n", res.Stdout)
}

func TestResolve_AnEmptyDirectoryIsNotACloneYet(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.Mkdir("projects/security-services/sec-audit-collector")

	located := home.Mur("repo", "locate", "collector", "~/projects/security-services/sec-audit-collector",
		"--workspace", "auditlogs").OK(t)
	res := home.Mur("resolve", "collector", "--workspace", "auditlogs").OK(t)

	require.Contains(t, located.Stdout, "cloned: false")
	require.Contains(t, res.Stdout, "cloned: false")
}
