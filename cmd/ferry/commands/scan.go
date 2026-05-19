package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/paths"
	"github.com/djleonskennedy/ferry/internal/scan"
)

func newScan() *cobra.Command {
	var (
		write bool
		prune bool
	)
	cmd := &cobra.Command{
		Use:   "scan",
		Short: "Re-scan for env files (dry run by default; --write to update ferry.toml)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := projectRoot()
			if err != nil {
				return err
			}
			cfg, err := config.LoadProject(paths.ProjectConfigPath(root))
			if err != nil {
				return err
			}
			candidates, err := scan.Discover(root, nil)
			if err != nil {
				return err
			}

			have := map[string]config.FileEntry{}
			for _, f := range cfg.Files {
				have[f.Path] = f
			}
			found := map[string]struct{}{}
			for _, c := range candidates {
				found[c.RelPath] = struct{}{}
			}

			var toAdd []string
			for _, c := range candidates {
				if _, ok := have[c.RelPath]; !ok {
					toAdd = append(toAdd, c.RelPath)
				}
			}
			var toRemove []string
			for _, f := range cfg.Files {
				abs := filepath.Join(root, filepath.FromSlash(f.Path))
				if _, err := os.Stat(abs); os.IsNotExist(err) {
					toRemove = append(toRemove, f.Path)
				}
			}
			sort.Strings(toAdd)
			sort.Strings(toRemove)

			out := cmd.OutOrStdout()
			if len(toAdd) == 0 && len(toRemove) == 0 {
				fmt.Fprintln(out, "No changes.")
				return nil
			}
			for _, p := range toAdd {
				fmt.Fprintf(out, "+ %s\n", p)
			}
			for _, p := range toRemove {
				fmt.Fprintf(out, "- %s\n", p)
			}

			if !write && !prune {
				fmt.Fprintln(out, "\n(dry run — pass --write to add new entries, --prune to remove missing)")
				return nil
			}
			if write {
				for _, p := range toAdd {
					cfg.Files = append(cfg.Files, config.FileEntry{Path: p, Required: true})
				}
			}
			if prune {
				kept := cfg.Files[:0]
				removed := map[string]struct{}{}
				for _, p := range toRemove {
					removed[p] = struct{}{}
				}
				for _, f := range cfg.Files {
					if _, drop := removed[f.Path]; drop {
						continue
					}
					kept = append(kept, f)
				}
				cfg.Files = kept
			}
			sort.Slice(cfg.Files, func(i, j int) bool { return cfg.Files[i].Path < cfg.Files[j].Path })
			return config.SaveProject(paths.ProjectConfigPath(root), cfg)
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "add newly discovered files to ferry.toml")
	cmd.Flags().BoolVar(&prune, "prune", false, "remove ferry.toml entries whose files no longer exist")
	return cmd
}

// projectRoot looks for ferry.toml walking up from cwd.
func projectRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return paths.FindProjectRoot(cwd)
}
