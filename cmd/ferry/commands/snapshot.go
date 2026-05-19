package commands

import (
	"fmt"
	"os"

	"filippo.io/age"
	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
	"github.com/djleonskennedy/ferry/internal/paths"
	"github.com/djleonskennedy/ferry/internal/snapshot"
)

func newSnapshot() *cobra.Command {
	var (
		keyID   string
		plain   bool
		message string
	)
	cmd := &cobra.Command{
		Use:     "snapshot",
		Aliases: []string{"snap"},
		Short:   "Create an encrypted snapshot of the configured files",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := projectRoot()
			if err != nil {
				return err
			}
			cfg, err := config.LoadProject(paths.ProjectConfigPath(root))
			if err != nil {
				return err
			}
			gcfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			if keyID == "" {
				keyID = cfg.Encryption.KeyID
			}

			var recipients []age.Recipient
			if !plain {
				path, err := gcfg.ResolveKeyPath(keyID)
				if err != nil {
					return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
				}
				id, err := crypto.LoadIdentity(path)
				if err != nil {
					return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
				}
				recipients = []age.Recipient{crypto.RecipientFromIdentity(id)}
			} else if cfg.Encryption.Required {
				return fmt.Errorf("%w: --plain refused (encryption.required = true)", cliutil.ErrAbort)
			}

			res, err := snapshot.Create(snapshot.CreateOpts{
				Root:         root,
				ProjectCfg:   cfg,
				Recipients:   recipients,
				AllowPlain:   plain,
				Message:      message,
				FerryVersion: buildVersion,
			})
			if err != nil {
				return err
			}
			if err := snapshot.Prune(cfg.Project.Name, gcfg.Defaults.SnapshotRetention); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Created snapshot v%d (%d files, key=%s)\n", res.Version, len(res.Manifest.Files), cfg.Encryption.KeyID)
			fmt.Fprintf(out, "  %s\n", res.Dir)
			return nil
		},
	}
	cmd.Flags().StringVar(&keyID, "key", "", "key id to encrypt with (defaults to encryption.key_id in ferry.toml)")
	cmd.Flags().BoolVar(&plain, "plain", false, "write a plaintext tar instead of encrypted (rejected unless encryption.required=false)")
	cmd.Flags().StringVarP(&message, "message", "m", "", "optional message recorded in the manifest")
	// help command testing aid: if FERRY_HOME is unset we still want command to fail clearly
	_ = os.Getenv
	return cmd
}
