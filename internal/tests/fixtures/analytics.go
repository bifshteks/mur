package fixtures

const (
	CollectorURL = "git@git.nimbus.example:nimbus/services-team/svc-analytics-collector.git"
	CollectorID  = "git.nimbus.example/nimbus/services-team/svc-analytics-collector"
	LogrotateURL = "git@git.nimbus.example:nimbus/services-team/svc-analytics-logrotate.git"
	LogrotateID  = "git.nimbus.example/nimbus/services-team/svc-analytics-logrotate"
)

// Analytics attaches source nimbus with an analytics workspace holding collector
// (aliased, with langs and status) and logrotate (bare), neither located yet.
func (h *Home) Analytics() {
	h.t.Helper()

	h.Source("nimbus")
	h.Mur("workspace", "create", "analytics", "--summary", "Analytics events of Nimbus Cloud.").OK(h.t)
	h.WriteFile("projects/nimbus/workspaces/analytics/workspace.yaml", `summary: Analytics events of Nimbus Cloud.
statuses:
  - name: live
    description: Ships, deployed or used.
  - name: dead
    description: Superseded. Do not change.
    sync: false
`)
	h.Mur("repo", "add", CollectorURL, "--workspace", "analytics",
		"--description", "Reads k8s node files and exports them.",
		"--alias", "collector", "--lang", "go", "--status", "live").OK(h.t)
	h.Mur("repo", "add", LogrotateURL, "--workspace", "analytics",
		"--description", "Rotates analytics log files on nodes.").OK(h.t)
}
