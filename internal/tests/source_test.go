package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestSourceInit_WritesSkeletonIntoAnExistingEmptyRepo(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	dir := home.EmptyRepo("projects/workspace")

	home.Mur("source", "init", "mws", "--path", dir).OK(t)

	murYAML, err := os.ReadFile(filepath.Join(dir, "mur.yaml"))
	require.NoError(t, err)
	require.Equal(t, "version: 1\n", string(murYAML))
	require.DirExists(t, filepath.Join(dir, "workspaces"))
	require.Equal(t, dir, home.Git(dir, "rev-parse", "--show-toplevel"), "no nested repo")

	list := home.Mur("source", "list").OK(t)
	require.YAMLEq(t, "- name: mws\n  path: "+dir+"\n", list.Stdout)
}

func TestSourceAdd_ClonesASharedSourceAndItsWorkspacesBecomeReadable(t *testing.T) {
	t.Parallel()

	author := fixtures.NewHome(t)
	author.Auditlogs()
	url := author.PublishSource(author.Path("projects/mws"), "mws")

	reader := fixtures.NewHome(t)
	reader.Mur("source", "add", "mws", url).OK(t)

	reader.Mur("resolve", "collector", "--workspace", "auditlogs").OK(t)
	list := reader.Mur("source", "list").OK(t)
	require.YAMLEq(t, "- name: mws\n  path: "+reader.Path(".config/mur/sources/mws")+"\n  remote: "+url+"\n", list.Stdout)
}

func TestSourceAdd_AnExistingCheckoutIsAttachedWhereItIs(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.Mur("source", "remove", "mws").OK(t)
	require.FileExists(t, home.Path("projects/mws/mur.yaml"), "remove never deletes files")
	require.Equal(t, cli.ExitNotFound, home.Mur("resolve", "collector", "--workspace", "auditlogs").Code)

	home.Mur("source", "add", "mws", "~/projects/mws").OK(t)

	home.Mur("resolve", "collector", "--workspace", "auditlogs").OK(t)
}

func TestSourcePull_BringsInWhatTeammatesPushed(t *testing.T) {
	t.Parallel()

	author := fixtures.NewHome(t)
	author.Auditlogs()
	url := author.PublishSource(author.Path("projects/mws"), "mws")
	reader := fixtures.NewHome(t)
	reader.Mur("source", "add", "mws", url).OK(t)

	author.Mur("workspace", "create", "modelhub").OK(t)
	author.Git(author.Path("projects/mws"), "add", ".")
	author.Git(author.Path("projects/mws"), "commit", "-q", "-m", "modelhub")
	author.Git(author.Path("projects/mws"), "push", "-q")

	res := reader.Mur("source", "pull").OK(t)

	require.YAMLEq(t, "- name: mws\n  result: pulled\n", res.Stdout)
	reader.Mur("workspace", "show", "modelhub").OK(t)
}

func TestSource_ANewerFormatIsRefusedWithAClearMessage(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()
	home.WriteFile("projects/mws/mur.yaml", "version: 2\n")

	res := home.Mur("workspace", "list")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "upgrade mur")
}
