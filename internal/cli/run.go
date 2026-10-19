package cli

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/git"
	"github.com/bifshteks/mur/internal/state"
)

// Env is everything mur reads from the outside world. main.go fills it from the
// process; tests fill it with a throwaway home so they can run in parallel.
type Env struct {
	Args    []string
	Stdin   io.Reader
	Stdout  io.Writer
	Stderr  io.Writer
	IsTTY   bool
	Environ []string
	Dir     string
}

func (e Env) Getenv(key string) string {
	for i := len(e.Environ) - 1; i >= 0; i-- {
		if k, v, ok := strings.Cut(e.Environ[i], "="); ok && k == key {
			return v
		}
	}

	return ""
}

// app is one invocation: the env, the output format and the state file.
type app struct {
	env  Env
	git  git.Git
	json bool
}

func Run(ctx context.Context, env Env) int {
	a := &app{env: env, git: git.Git{Environ: env.Environ}}
	root := a.rootCommand()
	root.SetArgs(env.Args)
	root.SetIn(env.Stdin)
	root.SetOut(env.Stdout)
	root.SetErr(env.Stderr)

	err := root.ExecuteContext(ctx)
	if err == nil {
		return ExitOK
	}

	var ee *exitError
	if !errors.As(err, &ee) {
		// Anything that did not come out of a command body is cobra rejecting
		// the command line itself.
		ee = &exitError{code: ExitUsage, err: err}
	}

	a.printError(ee)

	return ee.code
}

func (a *app) murHome() string {
	if h := a.env.Getenv("MUR_HOME"); h != "" {
		return h
	}

	if x := a.env.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "mur")
	}

	return filepath.Join(a.env.Getenv("HOME"), ".config", "mur")
}

func (a *app) statePath() string {
	return filepath.Join(a.murHome(), "state.yaml")
}

func (a *app) loadState() (*state.State, error) {
	return state.Load(a.statePath())
}

func (a *app) saveState(s *state.State) error {
	return s.Save(a.statePath())
}

// abs makes a user-supplied path absolute against the caller's directory,
// expanding a leading ~.
func (a *app) abs(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		p = filepath.Join(a.env.Getenv("HOME"), p[1:])
	}

	if !filepath.IsAbs(p) {
		p = filepath.Join(a.env.Dir, p)
	}

	return filepath.Clean(p)
}

func (a *app) rootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "mur",
		Short:         "Which repos a workspace has, what each is for, and where each is cloned",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.PersistentFlags().BoolVar(&a.json, "json", false, "print JSON instead of YAML")
	root.AddCommand(a.sourceCommand(), a.workspaceCommand(), a.repoCommand(), a.resolveCommand(), a.syncCommand())

	return root
}
