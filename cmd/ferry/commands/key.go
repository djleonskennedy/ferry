package commands

import "github.com/spf13/cobra"

func newKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage encryption keys",
	}
	cmd.AddCommand(newKeyGenerate(), newKeyList(), newKeyImport())
	return cmd
}
