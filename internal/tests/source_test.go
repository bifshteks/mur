package tests_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

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
