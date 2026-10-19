package fixtures

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		args := []string{
			"repo", "add", "file://" + remotes[r.Name], "--workspace", "team",
			"--description", r.Name + " service.", "--alias", r.Name,
		}

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

// RepoID is the id mur gives a local bare remote: its path without .git.
func RepoID(remote string) string {
	return strings.TrimSuffix(remote, ".git")
}

// EmptyRemote is a bare repo with no commits yet, like a project just created
// on GitLab.
func (h *Home) EmptyRemote(name string) string {
	h.t.Helper()

	bare := h.Path("remotes/" + name + ".git")
	require.NoError(h.t, os.MkdirAll(bare, 0o750))
	h.Git(bare, "init", "-q", "--bare")

	return bare
}

// HangingRemote makes every ssh remote hang forever, like a dead VPN. It
// returns a url to use and a channel that closes once git is stuck on it.
func (h *Home) HangingRemote() (url string, stuck <-chan struct{}) {
	h.t.Helper()

	marker := h.Path("ssh-started")
	script := h.WriteFile("bin/hanging-ssh", "#!/bin/sh\ntouch '"+marker+"'\nexec sleep 60\n")
	require.NoError(h.t, os.Chmod(script, 0o700)) //nolint:gosec // test helper script
	h.Setenv("GIT_SSH_COMMAND", script)

	started := make(chan struct{})

	go func() {
		defer close(started)

		for range 200 {
			if _, err := os.Stat(marker); err == nil {
				return
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()

	return "ssh://git@example.invalid/team/slow.git", started
}

// MurInterrupted runs mur, waits until git is stuck on a hanging remote, runs
// meanwhile (if any) while mur is still busy, then interrupts it as Ctrl-C
// would. It fails the test if mur does not stop promptly.
func (h *Home) MurInterrupted(stuck <-chan struct{}, meanwhile func(), args ...string) Result {
	h.t.Helper()

	ctx, interrupt := context.WithCancel(h.t.Context())
	defer interrupt()

	done := make(chan Result, 1)

	go func() { done <- h.runCtx(ctx, false, "", args) }()

	<-stuck

	if meanwhile != nil {
		meanwhile()
	}

	interrupt()

	select {
	case res := <-done:
		return res
	case <-time.After(20 * time.Second):
		h.t.Fatal("mur did not stop after the interrupt")

		return Result{}
	}
}
