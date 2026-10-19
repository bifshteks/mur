package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const waitDelay = 5 * time.Second

// Git runs the git binary, so SSH config, credentials and worktrees behave
// exactly as in the user's terminal.
type Git struct {
	Environ []string
}

func (g Git) Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = g.Environ
	// On cancel git gets SIGINT, not SIGKILL, so it removes a half-done clone
	// itself. WaitDelay bounds how long an ssh child may hold the pipes after.
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = waitDelay

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("git %s: %w", args[0], ctx.Err())
		}

		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}

		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsRepo reports whether dir is the top of a git working tree. git answers
// with symlinks resolved, so dir is resolved too before comparing.
func (g Git) IsRepo(ctx context.Context, dir string) bool {
	top, err := g.Run(ctx, dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return false
	}

	resolved, err := filepath.EvalSymlinks(dir)

	return err == nil && top == resolved
}

// Origin is the url of the origin remote, empty when there is none.
func (g Git) Origin(ctx context.Context, dir string) string {
	url, err := g.Run(ctx, dir, "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}

	return url
}
