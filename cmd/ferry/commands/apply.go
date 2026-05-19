package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
	"github.com/djleonskennedy/ferry/internal/paths"
	"github.com/djleonskennedy/ferry/internal/snapshot"
)

func newApply() *cobra.Command {
	var (
		version int
		force   bool
		noBack  bool
		toDir   string
	)
	cmd := &cobra.Command{
		Use:   "apply",
		Short: "Restore files from a snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := projectRoot()
			if err != nil {
				return err
			}
			dest := root
			if toDir != "" {
				dest = toDir
			}
			cfg, err := config.LoadProject(paths.ProjectConfigPath(root))
			if err != nil {
				return err
			}
			gcfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			keyPath, err := gcfg.ResolveKeyPath(cfg.Encryption.KeyID)
			if err != nil {
				return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
			}
			id, err := crypto.LoadIdentity(keyPath)
			if err != nil {
				return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
			}
			backupOn := gcfg.Defaults.BackupOnApply && !noBack

			res, err := snapshot.Apply(snapshot.ApplyOpts{
				Project:       cfg.Project.Name,
				Dest:          dest,
				Identity:      id,
				Version:       version,
				Force:         force,
				BackupOnForce: backupOn,
				BackupRetain:  gcfg.Defaults.BackupRetention,
			})
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Applied snapshot v%d to %s\n", res.Version, dest)
			fmt.Fprintf(out, "  restored: %d\n", len(res.Restored))
			fmt.Fprintf(out, "  skipped (already same): %d\n", len(res.SkippedSame))
			if len(res.BackedUp) > 0 {
				fmt.Fprintf(out, "  backed up %d file(s) to %s\n", len(res.BackedUp), res.BackupDir)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&version, "version", 0, "snapshot version (0 = latest)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite locally-modified files (creates backup first)")
	cmd.Flags().BoolVar(&noBack, "no-backup", false, "disable pre-overwrite backup (used with --force)")
	cmd.Flags().StringVar(&toDir, "to", "", "restore into this directory instead of the project root")
	return cmd
}
