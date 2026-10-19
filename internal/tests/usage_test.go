package tests_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bifshteks/mur/internal/cli"
	"github.com/bifshteks/mur/internal/tests/fixtures"
)

func TestAMistypedSubcommandIsAUsageErrorNotASilentSuccess(t *testing.T) {
	t.Parallel()

	home := fixtures.NewHome(t)
	home.Auditlogs()

	res := home.Mur("workspace", "activat", "auditlogs")

	require.Equal(t, cli.ExitUsage, res.Code)
	require.Contains(t, res.Stderr, `unknown command "activat"`)
}
