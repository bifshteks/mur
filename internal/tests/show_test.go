package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestWorkspaceShow_GivesTheReadmeReposAndExtraFilesToOpenOnDemand(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.WriteFile("projects/mws/workspaces/auditlogs/README.md", "# auditlogs\n\nCollector -> Kafka -> Iceberg.\n")
	home.WriteFile("projects/mws/workspaces/auditlogs/glossary.md", "# Glossary\n")
	home.WriteFile("projects/mws/workspaces/auditlogs/notes/iam.md", "# IAM\n")
	home.Mur("repo", "locate", "collector", "/srv/collector", "--workspace", "auditlogs").OK(t)
	ws := home.Path("projects/mws/workspaces/auditlogs")

	res := home.Mur("workspace", "show", "auditlogs").OK(t)

	require.YAMLEq(t, `
name: auditlogs
source: mws
dir: `+ws+`
summary: Audit events of MWS Cloud.
readme: |
  # auditlogs

  Collector -> Kafka -> Iceberg.
statuses:
  - name: live
    description: Ships, deployed or used.
  - name: dead
    description: Superseded. Do not change.
    sync: false
repos:
  - id: git.mws-team.ru/mws/security-services/sec-audit-collector
    url: git@git.mws-team.ru:mws/security-services/sec-audit-collector.git
    path: /srv/collector
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
files:
  - `+ws+`/glossary.md
  - `+ws+`/notes/iam.md
`, res.Stdout)
}
