package fixtures

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/stretchr/testify/require"
)

// Remote is a bare repo with one commit on master, standing in for GitLab.
func (h *Home) Remote(name string) string {
	h.t.Helper()

	bare := h.Path("remotes/" + name + ".git")
	require.NoError(h.t, os.MkdirAll(bare, 0o750))
	h.Git(bare, "init", "-q", "--bare")
	h.PushCommit(bare, "initial commit")

	return bare
}

// PushCommit lands a new commit on the remote's master, as a teammate would.
func (h *Home) PushCommit(remote, msg string) {
	h.t.Helper()

	work, err := os.MkdirTemp(h.Dir, "teammate-")
	require.NoError(h.t, err)

	defer os.RemoveAll(work)

	h.Git(h.Dir, "clone", "-q", remote, work)
	f := filepath.Join(work, strings.ReplaceAll(msg, " ", "-")+".txt")
	require.NoError(h.t, os.WriteFile(f, []byte(msg+"\n"), 0o600))
	h.Git(work, "add", ".")
	h.Git(work, "commit", "-q", "-m", msg)
	h.Git(work, "push", "-q", "origin", "HEAD:master")
}

// Clone checks a remote out under this home, as the user once did by hand.
func (h *Home) Clone(remote, rel string) string {
	h.t.Helper()

	dir := h.Path(rel)
	h.Git(h.Dir, "clone", "-q", remote, dir)

	return dir
}

// Team attaches source mws with a workspace "team" declaring statuses live and
// dead (sync: false), with one repo per name, each backed by a bare remote
// under ~/remotes and spelled as a file:// url. It returns the remotes by name.
func (h *Home) Team(repos ...TeamRepo) map[string]string {
	h.t.Helper()

	h.Source("mws")
	h.Mur("workspace", "create", "team").OK(h.t)
	h.WriteFile("projects/mws/workspaces/team/workspace.yaml", `summary: A team with real remotes.
statuses:
  - name: live
    description: Deployed.
  - name: dead
    description: Superseded.
    sync: false
`)

	remotes := map[string]string{}

	for _, r := range repos {
		remotes[r.Name] = h.Remote(r.Name)
		args := []string{"repo", "add", "file://" + remotes[r.Name], "--workspace", "team",
			"--description", r.Name + " service.", "--alias", r.Name}

		if r.Status != "" {
			args = append(args, "--status", r.Status)
		}

		h.Mur(args...).OK(h.t)
	}

	return remotes
}

type TeamRepo struct {
	Name   string
	Status string
}

// PublishSource commits a local source and pushes it to a fresh bare remote
// under ~/remotes, as the user does after editing workspaces. It returns the
// remote's url.
func (h *Home) PublishSource(dir, name string) string {
	h.t.Helper()

	bare := h.Path("remotes/" + name + "-source.git")
	require.NoError(h.t, os.MkdirAll(bare, 0o750))
	h.Git(bare, "init", "-q", "--bare")
	h.Git(dir, "add", ".")
	h.Git(dir, "commit", "-q", "-m", "workspaces")
	h.Git(dir, "remote", "add", "origin", bare)
	h.Git(dir, "push", "-q", "-u", "origin", "HEAD:master")

	return bare
}
