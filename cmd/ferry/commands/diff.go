package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/paths"
	"github.com/djleonskennedy/ferry/internal/snapshot"
)

func newDiff() *cobra.Command {
	var version int
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show drift status between current files and a snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := projectRoot()
			if err != nil {
				return err
			}
			cfg, err := config.LoadProject(paths.ProjectConfigPath(root))
			if err != nil {
				return err
			}
			res, err := snapshot.Diff(snapshot.DiffOpts{
				Project: cfg.Project.Name,
				Dest:    root,
				Version: version,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			rows := make([][]string, 0, len(res.Files))
			for _, f := range res.Files {
				rows = append(rows, []string{f.Status.String(), f.Path})
			}
			cliutil.PrintTable(out, []string{"STATUS", "PATH"}, rows)
			if res.HasDrift() {
				return fmt.Errorf("%w: snapshot v%d differs from disk", cliutil.ErrDrift, res.Version)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&version, "version", 0, "snapshot version (0 = latest)")
	return cmd
}
