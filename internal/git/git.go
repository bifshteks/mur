package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Git runs the git binary, so SSH config, credentials and worktrees behave
// exactly as in the user's terminal.
type Git struct {
	Environ []string
}

func (g Git) Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = g.Environ

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}

		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsRepo reports whether dir is the top of a git working tree.
func (g Git) IsRepo(ctx context.Context, dir string) bool {
	top, err := g.Run(ctx, dir, "rev-parse", "--show-toplevel")

	return err == nil && top == dir
}

// Origin is the url of the origin remote, empty when there is none.
func (g Git) Origin(ctx context.Context, dir string) string {
	url, err := g.Run(ctx, dir, "config", "--get", "remote.origin.url")
	if err != nil {
		return ""
	}

	return url
}
