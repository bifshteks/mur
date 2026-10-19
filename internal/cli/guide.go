package cli

import (
	"embed"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// The examples are real mur output, checked byte for byte by the acceptance
// tests, so the guide cannot drift from the binary.
//
//go:embed guide
var guideExamples embed.FS

const guideIntro = `# mur — agent guide

mur knows, for a named workspace, which repos it has, what each is for, and
where each is cloned on this laptop. Ask it instead of guessing paths or
searching the disk.

## How to use it

1. Before working across a workspace, read it: mur workspace show [name].
   Its README describes the system; files: lists extra docs to open on demand.
2. For each repo you touch, run mur resolve <alias|basename|url>... and work
   in the path it returns. --path prints only absolute paths, for $(...).
3. Never guess a path. Exit 3 (needs_path) means mur does not know where a
   repo is: ask the user, then record it with mur repo locate.

Reads take the workspace from --workspace, then $MUR_WORKSPACE, then the
active one (mur workspace activate). Writes (repo add, repo remove) require
--workspace. Set MUR_WORKSPACE to pin a workspace for one session.

Output is YAML; --json prints the same data as JSON. mur never prompts when
stdin is not a terminal, and never commits or pushes: edits to a source are
plain files for the user to commit.

## Exit codes

- 0 ok
- 1 error (one repo failing in sync does not stop the others)
- 2 usage: bad flags or arguments, or no workspace selected
- 3 needs input: needs_path lists repos with no known path
- 4 not found or ambiguous: candidates are printed on stderr
`

func (a *app) agentGuideCommand(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "agent-guide",
		Short: "Print the manual for LLM agents: commands, output shapes, exit codes",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			guide, err := renderGuide(root)
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(a.env.Stdout, guide)

			return err
		},
	}
}

func renderGuide(root *cobra.Command) (string, error) {
	var b strings.Builder

	b.WriteString(guideIntro)
	b.WriteString("\n## Commands\n\nGlobal: --json. Environment: MUR_HOME, MUR_WORKSPACE.\n\n")
	writeCommands(&b, root)

	b.WriteString("\n## Output\n")

	for _, ex := range []struct{ title, file string }{
		{"mur resolve collector", "resolve.yaml"},
		{"mur resolve collector --path", "resolve-path.txt"},
		{"mur sync, without a terminal, one repo it cannot place (exit 3)", "sync.yaml"},
		{"mur resolve exporter, on stderr (exit 4)", "not-found.yaml"},
	} {
		data, err := guideExamples.ReadFile("guide/" + ex.file)
		if err != nil {
			return "", fmt.Errorf("read guide example: %w", err)
		}

		fmt.Fprintf(&b, "\n%s:\n\n```\n%s```\n", ex.title, data)
	}

	return b.String(), nil
}

// writeCommands lists every runnable command with its arguments and flags,
// straight from the command tree.
func writeCommands(b *strings.Builder, cmd *cobra.Command) {
	for _, c := range cmd.Commands() {
		if c.Hidden || c.Name() == "help" {
			continue
		}

		if c.Runnable() && !c.HasSubCommands() {
			path := strings.TrimPrefix(c.CommandPath(), "mur ")
			args := strings.TrimPrefix(c.Use, c.Name())
			fmt.Fprintf(b, "mur %s%s — %s\n", path, args, c.Short)

			if flags := c.LocalNonPersistentFlags().FlagUsages(); flags != "" {
				b.WriteString(flags)
			}

			b.WriteString("\n")
		}

		writeCommands(b, c)
	}
}
