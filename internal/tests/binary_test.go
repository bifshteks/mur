package tests_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
)

// One run of the real binary: main.go's wiring of streams, version stamping
// and exit codes is the only part the in-process tests cannot reach.
func TestBinary_ExitCodesAndVersionReachTheShell(t *testing.T) {
	t.Parallel()

	bin := filepath.Join(t.TempDir(), "mur")
	build := exec.CommandContext(t.Context(), "go", "build",
		"-ldflags", "-X main.version=v9.9.9 -X main.buildTime=2026-10-19T00:00:00Z",
		"-o", bin, "github.com/bifshteks/mur")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "%s", out)

	run := func(args ...string) (string, int) {
		cmd := exec.CommandContext(t.Context(), bin, args...)
		cmd.Env = []string{"HOME=" + t.TempDir()}
		out, err := cmd.Output()

		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return string(out), exit.ExitCode()
		}

		require.NoError(t, err)

		return string(out), 0
	}

	version, code := run("--version")
	require.Equal(t, 0, code)
	require.Equal(t, "mur v9.9.9 (built 2026-10-19T00:00:00Z)\n", version)

	_, code = run("resolve")
	require.Equal(t, cli.ExitUsage, code, "missing argument")

	_, code = run("no-such-command")
	require.Equal(t, cli.ExitUsage, code)

	_, code = run("resolve", "collector", "--workspace", "auditlogs")
	require.Equal(t, cli.ExitNotFound, code, "no such workspace")
}

// An agent harness runs mur with stdin from /dev/null: a character device, but
// not a terminal. mur must not prompt there.
func TestBinary_StdinFromDevNullIsNotATerminal(t *testing.T) {
	t.Parallel()

	bin := filepath.Join(t.TempDir(), "mur")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, "github.com/bifshteks/mur")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "%s", out)

	home := t.TempDir()
	run := func(args ...string) (string, int) {
		devNull, err := os.Open(os.DevNull)
		require.NoError(t, err)

		defer devNull.Close()

		cmd := exec.CommandContext(t.Context(), bin, args...)
		cmd.Env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1"}
		cmd.Stdin = devNull

		var stderr strings.Builder
		cmd.Stderr = &stderr

		err = cmd.Run()

		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return stderr.String(), exit.ExitCode()
		}

		require.NoError(t, err, stderr.String())

		return stderr.String(), 0
	}

	run("source", "init", "mws")
	run("workspace", "create", "team")
	run("repo", "add", "git@example.invalid:team/gateway.git", "--workspace", "team", "--description", "Gateway.")

	stderr, code := run("sync", "--workspace", "team")

	require.Equal(t, cli.ExitNeedsPath, code)
	require.NotContains(t, stderr, "has no path")
}
