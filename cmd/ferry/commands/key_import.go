package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
	"github.com/djleonskennedy/ferry/internal/paths"
)

func newKeyImport() *cobra.Command {
	var (
		id    string
		from  string
		force bool
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import an existing age key",
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return fmt.Errorf("%w: --from is required", cliutil.ErrUsage)
			}
			// Validate the source first.
			if _, err := crypto.LoadIdentity(from); err != nil {
				return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
			}
			dst := paths.KeyFile(id)
			if _, err := os.Stat(dst); err == nil && !force {
				return fmt.Errorf("%w: key %s already exists at %s (use --force)", cliutil.ErrUsage, id, dst)
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
				return err
			}
			if err := copyFile0600(from, dst); err != nil {
				return fmt.Errorf("%w: %v", cliutil.ErrKey, err)
			}
			gcfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			gcfg.Keys[id] = config.KeyEntry{Path: dst}
			if err := config.SaveGlobal(gcfg); err != nil {
				return err
			}
			ident, _ := crypto.LoadIdentity(dst)
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Imported key %s at %s\n", id, dst)
			if ident != nil {
				fmt.Fprintf(out, "Recipient (public): %s\n", crypto.RecipientString(ident))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "default", "key id")
	cmd.Flags().StringVar(&from, "from", "", "path to existing key file")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing key")
	return cmd
}

func copyFile0600(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
