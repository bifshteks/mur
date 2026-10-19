package cli

import (
	"strings"

	"github.com/spf13/cobra"
)

const keyScanRoots = "scan_roots"

var defaultScanRoots = []string{"~/projects"} //nolint:gochecknoglobals // a default, never mutated

func (a *app) configCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read or change local settings (scan_roots)",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "get scan_roots",
		Short: "Print a setting",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if args[0] != keyScanRoots {
				return usagef("unknown setting %s; known: %s", args[0], keyScanRoots)
			}

			st, err := a.loadState()
			if err != nil {
				return fail(err)
			}

			roots := st.ScanRoots
			if len(roots) == 0 {
				roots = defaultScanRoots
			}

			return fail(a.print(roots))
		},
	}, &cobra.Command{
		Use:   "set scan_roots <dir>[,<dir>...]",
		Short: "Change a setting",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			if args[0] != keyScanRoots {
				return usagef("unknown setting %s; known: %s", args[0], keyScanRoots)
			}

			st, err := a.loadState()
			if err != nil {
				return fail(err)
			}

			st.ScanRoots = nil

			for _, r := range strings.Split(args[1], ",") {
				if r = strings.TrimSpace(r); r != "" {
					st.ScanRoots = append(st.ScanRoots, r)
				}
			}

			if err := a.saveState(st); err != nil {
				return fail(err)
			}

			return fail(a.print(st.ScanRoots))
		},
	})

	return cmd
}
