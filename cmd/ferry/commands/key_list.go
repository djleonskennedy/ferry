package commands

import (
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/djleonskennedy/ferry/internal/cliutil"
	"github.com/djleonskennedy/ferry/internal/config"
	"github.com/djleonskennedy/ferry/internal/crypto"
)

func newKeyList() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List configured keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			gcfg, err := config.LoadGlobal()
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(gcfg.Keys) == 0 {
				fmt.Fprintln(out, "No keys configured. Run `ferry key generate`.")
				return nil
			}
			ids := make([]string, 0, len(gcfg.Keys))
			for id := range gcfg.Keys {
				ids = append(ids, id)
			}
			sort.Strings(ids)
			header := []string{"ID", "PATH", "MODE", "RECIPIENT"}
			rows := make([][]string, 0, len(ids))
			for _, id := range ids {
				path, _ := gcfg.ResolveKeyPath(id)
				mode := "—"
				recipient := "—"
				if fi, err := os.Stat(path); err == nil {
					perm := fi.Mode().Perm()
					mode = fmt.Sprintf("%04o", perm)
					if perm&0o077 != 0 {
						mode += " (!)"
					}
				}
				if ident, err := crypto.LoadIdentity(path); err == nil {
					recipient = crypto.RecipientString(ident)
				}
				rows = append(rows, []string{id, path, mode, recipient})
			}
			cliutil.PrintTable(out, header, rows)
			return nil
		},
	}
	return cmd
}
