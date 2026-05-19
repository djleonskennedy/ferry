package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/paths"
	"github.com/djleonskennedy/ferry/internal/scan"
)

func newInit() *cobra.Command {
	var (
		name  string
		force bool
	)
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Discover env files and write ferry.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			projectName := name
			if projectName == "" {
				projectName = filepath.Base(cwd)
			}
			cfgPath := paths.ProjectConfigPath(cwd)
			if _, err := os.Stat(cfgPath); err == nil && !force {
				return fmt.Errorf("%w: ferry.toml already exists; use --force to overwrite", cliutil.ErrUsage)
			}
			candidates, err := scan.Discover(cwd, nil)
			if err != nil {
				return err
			}
			cfg := config.DefaultProject(projectName)
			for _, c := range candidates {
				cfg.Files = append(cfg.Files, config.FileEntry{Path: c.RelPath, Required: true})
			}
			if err := config.SaveProject(cfgPath, cfg); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created %s with %d file(s):\n", cfgPath, len(candidates))
			for _, c := range candidates {
				fmt.Fprintf(out, "  %s\n", c.RelPath)
			}
			if len(candidates) == 0 {
				fmt.Fprintln(out, "  (none — add entries manually or run `ferry scan --write` later)")
			}
			fmt.Fprintln(out, "Review ferry.toml and remove anything you do not want to snapshot.")
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "project name (defaults to current directory name)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing ferry.toml")
	return cmd
}
