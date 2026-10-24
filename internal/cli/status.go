package cli

import (
	"github.com/spf13/cobra"

	"github.com/bifshteks/mur/internal/source"
)

func (a *app) statusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Declare the statuses a workspace's repos may have",
	}
	cmd.AddCommand(a.statusAddCommand())

	return cmd
}

func (a *app) statusAddCommand() *cobra.Command {
	var (
		ws   string
		s    source.Status
		sync bool
	)

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Declare a status in a workspace (writes workspace.yaml; --workspace required)",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if ws == "" {
				return usagef("status add writes a source; pass --workspace")
			}

			s.Name = args[0]
			if !sync {
				s.Sync = &sync
			}

			return a.statusAdd(ws, s)
		},
	}
	cmd.Flags().StringVar(&ws, "workspace", "", "workspace to declare the status in")
	cmd.Flags().StringVar(&s.Description, "description", "", "what a repo with this status is (required)")
	cmd.Flags().BoolVar(&sync, "sync", true, "whether sync touches repos with this status by default; --sync=false to skip them")
	_ = cmd.MarkFlagRequired("description")

	return cmd
}

func (a *app) statusAdd(ws string, s source.Status) error {
	w, _, err := a.findWorkspace(ws)
	if err != nil {
		return err
	}

	a.warnUnknown(w)

	if err := w.AddStatus(s); err != nil {
		return err
	}

	return a.print(w.Statuses[len(w.Statuses)-1])
}
