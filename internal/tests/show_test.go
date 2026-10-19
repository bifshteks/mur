package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestWorkspaceShow_GivesTheReadmeReposAndExtraFilesToOpenOnDemand(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.WriteFile("projects/nimbus/workspaces/analytics/README.md", "# analytics\n\nCollector -> Kafka -> Iceberg.\n")
	home.WriteFile("projects/nimbus/workspaces/analytics/glossary.md", "# Glossary\n")
	home.WriteFile("projects/nimbus/workspaces/analytics/notes/iam.md", "# IAM\n")
	home.Mur("repo", "locate", "collector", "/srv/collector", "--workspace", "analytics").OK(t)
	ws := home.Path("projects/nimbus/workspaces/analytics")

	res := home.Mur("workspace", "show", "analytics").OK(t)

	require.YAMLEq(t, `
name: analytics
source: nimbus
dir: `+ws+`
summary: Analytics events of Nimbus Cloud.
readme: |
  # analytics

  Collector -> Kafka -> Iceberg.
statuses:
  - name: live
    description: Ships, deployed or used.
  - name: dead
    description: Superseded. Do not change.
    sync: false
repos:
  - id: git.nimbus.example/nimbus/services-team/svc-analytics-collector
    url: git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git
    path: /srv/collector
    cloned: false
    description: Reads k8s node files and exports them.
    aliases:
      - collector
    langs:
      - go
    status: live
  - id: git.nimbus.example/nimbus/services-team/svc-analytics-logrotate
    url: git@git.nimbus.example:nimbus/services-team/svc-analytics-logrotate.git
    cloned: false
    description: Rotates analytics log files on nodes.
files:
  - `+ws+`/glossary.md
  - `+ws+`/notes/iam.md
`, res.Stdout)
}
