package fixtures

const (
	CollectorURL = "git@git.mws-team.ru:mws/security-services/sec-audit-collector.git"
	CollectorID  = "git.mws-team.ru/mws/security-services/sec-audit-collector"
	LogrotateURL = "git@git.mws-team.ru:mws/security-services/sec-audit-logrotate.git"
	LogrotateID  = "git.mws-team.ru/mws/security-services/sec-audit-logrotate"
)

// Auditlogs attaches source mws with an auditlogs workspace holding collector
// (aliased, with langs and status) and logrotate (bare), neither located yet.
func (h *Home) Auditlogs() {
	h.t.Helper()

	h.Source("mws")
	h.Mur("workspace", "create", "auditlogs", "--summary", "Audit events of MWS Cloud.").OK(h.t)
	h.WriteFile("projects/mws/workspaces/auditlogs/workspace.yaml", `summary: Audit events of MWS Cloud.
statuses:
  - name: live
    description: Ships, deployed or used.
  - name: dead
    description: Superseded. Do not change.
    sync: false
`)
	h.Mur("repo", "add", CollectorURL, "--workspace", "auditlogs",
		"--description", "Reads k8s node files and exports them.",
		"--alias", "collector", "--lang", "go", "--status", "live").OK(h.t)
	h.Mur("repo", "add", LogrotateURL, "--workspace", "auditlogs",
		"--description", "Rotates audit log files on nodes.").OK(h.t)
}
