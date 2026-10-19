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

	home.Mur("source", "init", "nimbus", "--path", dir).OK(t)

	murYAML, err := os.ReadFile(filepath.Join(dir, "mur.yaml"))
	require.NoError(t, err)
	require.Equal(t, "version: 1\n", string(murYAML))
	require.DirExists(t, filepath.Join(dir, "workspaces"))
	require.Equal(t, dir, home.Git(dir, "rev-parse", "--show-toplevel"), "no nested repo")

	list := home.Mur("source", "list").OK(t)
	require.YAMLEq(t, "- name: nimbus\n  path: "+dir+"\n", list.Stdout)
}

func TestSourceAdd_ClonesASharedSourceAndItsWorkspacesBecomeReadable(t *testing.T) {
	t.Parallel()

	author := fixtures.NewHome(t)
	author.Analytics()
	url := author.PublishSource(author.Path("projects/nimbus"), "nimbus")

	reader := fixtures.NewHome(t)
	reader.Mur("source", "add", "nimbus", url).OK(t)

	reader.Mur("resolve", "collector", "--workspace", "analytics").OK(t)
	list := reader.Mur("source", "list").OK(t)
	require.YAMLEq(t, "- name: nimbus\n  path: "+reader.Path(".config/mur/sources/nimbus")+"\n  remote: "+url+"\n", list.Stdout)
}

func TestSourceAdd_AnExistingCheckoutIsAttachedWhereItIs(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.Mur("source", "remove", "nimbus").OK(t)
	require.FileExists(t, home.Path("projects/nimbus/mur.yaml"), "remove never deletes files")
	require.Equal(t, cli.ExitNotFound, home.Mur("resolve", "collector", "--workspace", "analytics").Code)

	home.Mur("source", "add", "nimbus", "~/projects/nimbus").OK(t)

	home.Mur("resolve", "collector", "--workspace", "analytics").OK(t)
}

func TestSourcePull_BringsInWhatTeammatesPushed(t *testing.T) {
	t.Parallel()

	author := fixtures.NewHome(t)
	author.Analytics()
	url := author.PublishSource(author.Path("projects/nimbus"), "nimbus")
	reader := fixtures.NewHome(t)
	reader.Mur("source", "add", "nimbus", url).OK(t)

	author.Mur("workspace", "create", "modelhub").OK(t)
	author.Git(author.Path("projects/nimbus"), "add", ".")
	author.Git(author.Path("projects/nimbus"), "commit", "-q", "-m", "modelhub")
	author.Git(author.Path("projects/nimbus"), "push", "-q")

	res := reader.Mur("source", "pull").OK(t)

	require.YAMLEq(t, "- name: nimbus\n  result: pulled\n", res.Stdout)
	reader.Mur("workspace", "show", "modelhub").OK(t)
}

func TestSource_ANewerFormatIsRefusedWithAClearMessage(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	home.WriteFile("projects/nimbus/mur.yaml", "version: 2\n")

	res := home.Mur("workspace", "list")

	require.Equal(t, cli.ExitError, res.Code)
	require.Contains(t, res.Stderr, "upgrade mur")
}
