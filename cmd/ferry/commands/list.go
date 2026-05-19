package commands

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/paths"
	"github.com/djleonskennedy/ferry/internal/snapshot"
)

func newList() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available snapshots",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := projectRoot()
			if err != nil {
				return err
			}
			cfg, err := config.LoadProject(paths.ProjectConfigPath(root))
			if err != nil {
				return err
			}
			entries, err := snapshot.List(cfg.Project.Name)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(entries) == 0 {
				fmt.Fprintf(out, "No snapshots for %s yet — run `ferry snapshot`.\n", cfg.Project.Name)
				return nil
			}
			header := []string{"VERSION", "CREATED", "KEY", "FILES", "LATEST", "MESSAGE"}
			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				latest := ""
				if e.IsLatest {
					latest = "*"
				}
				rows = append(rows, []string{
					"v" + strconv.Itoa(e.Version),
					e.CreatedAt.Local().Format(time.RFC3339),
					e.KeyID,
					strconv.Itoa(e.FileCount),
					latest,
					e.Message,
				})
			}
			cliutil.PrintTable(out, header, rows)
			return nil
		},
	}
	return cmd
}
