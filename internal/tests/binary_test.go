package tests_test

import (
	"errors"
	"os/exec"
	"path/filepath"
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

	_, code = run("resolve", "collector", "--workspace", "analytics")
	require.Equal(t, cli.ExitNotFound, code, "no such workspace")
}
