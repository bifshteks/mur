package fixtures

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
)

// Home is a throwaway user: its own HOME, git identity and mur state, so tests
// never see the developer's config and can run in parallel.
type Home struct {
	t   *testing.T
	Dir string
	env []string
}

func NewHome(t *testing.T) *Home {
	t.Helper()

	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)

	gitconfig := "[user]\n\tname = Test User\n\temail = me@example.com\n" +
		"[init]\n\tdefaultBranch = master\n" +
		"[protocol \"file\"]\n\tallow = always\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".gitconfig"), []byte(gitconfig), 0o600))

	return &Home{
		t:   t,
		Dir: dir,
		env: []string{
			"HOME=" + dir,
			"PATH=" + os.Getenv("PATH"),
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
		},
	}
}

// Path is a path under this home.
func (h *Home) Path(rel string) string {
	return filepath.Join(h.Dir, rel)
}

// Setenv adds a variable to what mur and git see.
func (h *Home) Setenv(key, value string) {
	h.env = append(h.env, key+"="+value)
}

type Result struct {
	Stdout string
	Stderr string
	Code   int
}

// OK fails the test unless mur exited 0.
func (r Result) OK(t *testing.T) Result {
	t.Helper()
	require.Equal(t, 0, r.Code, "stderr: %s\nstdout: %s", r.Stderr, r.Stdout)

	return r
}

// Mur runs mur non-interactively, as an LLM harness would.
func (h *Home) Mur(args ...string) Result {
	h.t.Helper()

	return h.run(false, "", args)
}

// MurTTY runs mur as a person at a terminal typing input.
func (h *Home) MurTTY(input string, args ...string) Result {
	h.t.Helper()

	return h.run(true, input, args)
}

func (h *Home) run(tty bool, input string, args []string) Result {
	var stdout, stderr bytes.Buffer

	code := cli.Run(h.t.Context(), cli.Env{
		Args:    args,
		Stdin:   strings.NewReader(input),
		Stdout:  &stdout,
		Stderr:  &stderr,
		IsTTY:   tty,
		Environ: h.env,
		Dir:     h.Dir,
	})

	return Result{Stdout: stdout.String(), Stderr: stderr.String(), Code: code}
}

// Git runs git in dir with this home's identity and fails the test on error.
func (h *Home) Git(dir string, args ...string) string {
	h.t.Helper()

	//nolint:gosec // args are written by the test itself
	cmd := exec.CommandContext(h.t.Context(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = h.env
	out, err := cmd.CombinedOutput()
	require.NoError(h.t, err, "git %v: %s", args, out)

	return strings.TrimSpace(string(out))
}

// EmptyRepo is a freshly initialized git repo with no commits, like a project
// just created on GitLab and cloned.
func (h *Home) EmptyRepo(rel string) string {
	h.t.Helper()

	dir := h.Path(rel)
	require.NoError(h.t, os.MkdirAll(dir, 0o750))
	h.Git(dir, "init", "-q")

	return dir
}

// WriteFile writes a file under this home, creating its parents.
func (h *Home) WriteFile(rel, content string) string {
	h.t.Helper()

	p := h.Path(rel)
	require.NoError(h.t, os.MkdirAll(filepath.Dir(p), 0o750))
	require.NoError(h.t, os.WriteFile(p, []byte(content), 0o600))

	return p
}

// ReadFile reads a file under this home.
func (h *Home) ReadFile(rel string) string {
	h.t.Helper()

	data, err := os.ReadFile(h.Path(rel))
	require.NoError(h.t, err)

	return string(data)
}

// Source attaches a fresh source checked out at projects/<name>.
func (h *Home) Source(name string) string {
	h.t.Helper()

	dir := h.EmptyRepo("projects/" + name)
	h.Mur("source", "init", name, "--path", dir).OK(h.t)

	return dir
}
