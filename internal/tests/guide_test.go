package tests_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestAgentGuide_NamesEveryCommandOfTheContract(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)

	guide := home.Mur("agent-guide").OK(t).Stdout

	for _, cmd := range []string{
		"source init", "source add", "source list", "source pull", "source remove",
		"workspace list", "workspace create", "workspace activate", "workspace show",
		"repo add", "repo list", "repo remove", "repo locate",
		"resolve", "sync", "state push", "state pull", "config get", "config set",
	} {
		require.Contains(t, guide, "mur "+cmd+" ", "guide must document %q", cmd)
	}

	for _, s := range []string{"MUR_WORKSPACE", "--json", "needs_path"} {
		require.Contains(t, guide, s)
	}
}

// The guide's example outputs are what mur really prints, byte for byte, with
// the throwaway home spelled /home/me.
func TestAgentGuide_ExamplesAreRealOutput(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Analytics()
	collector := home.EmptyRepo("projects/services-team/svc-analytics-collector")
	home.Git(collector, "remote", "add", "origin", fixtures.CollectorURL)

	sync := home.Mur("sync", "--workspace", "analytics")
	require.Equal(t, cli.ExitNeedsPath, sync.Code)

	resolve := home.Mur("resolve", "collector", "--workspace", "analytics").OK(t)
	resolvePath := home.Mur("resolve", "collector", "--workspace", "analytics", "--path").OK(t)
	miss := home.Mur("resolve", "exporter", "--workspace", "analytics")
	require.Equal(t, cli.ExitNotFound, miss.Code)

	guide := home.Mur("agent-guide").OK(t).Stdout

	for name, out := range map[string]string{
		"sync":           sync.Stdout,
		"resolve":        resolve.Stdout,
		"resolve --path": resolvePath.Stdout,
		"not found":      miss.Stderr,
	} {
		require.Contains(t, guide, strings.ReplaceAll(out, home.Dir, "/home/me"), "example %q drifted from real output", name)
	}
}
