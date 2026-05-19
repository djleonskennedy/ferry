package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
	"github.com/djleonskennedy/ferry/internal/paths"
)

func newKeyGenerate() *cobra.Command {
	var (
		id    string
		force bool
	)
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Create a new age encryption key",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := paths.KeyFile(id)
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%w: key %s already exists at %s (use --force)", cliutil.ErrUsage, id, path)
			}
			ident, err := crypto.GenerateIdentity()
			if err != nil {
				return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
			}
			if err := crypto.WriteIdentity(path, ident); err != nil {
				return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
			}
			gcfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			gcfg.Keys[id] = config.KeyEntry{Path: path}
			if err := config.SaveGlobal(gcfg); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Generated key %s at %s\n", id, path)
			fmt.Fprintf(out, "Recipient (public): %s\n", crypto.RecipientString(ident))
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "default", "key id")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing key")
	return cmd
}
