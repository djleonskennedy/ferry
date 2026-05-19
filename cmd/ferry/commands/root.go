// Package commands wires ferry's cobra command tree.
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
	buildDate    = "unknown"
)

// SetBuildInfo is called from main with -ldflags-injected values.
func SetBuildInfo(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
}

// NewRoot builds the root command tree. Re-creates state-free on every call
// so tests can run commands.NewRoot().SetArgs(...).Execute() in isolation.
func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "ferry",
		Short:         "Snapshot, encrypt, and restore .env files",
		Long:          "ferry snapshots gitignored env files and restores them across worktrees and machines.",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", buildVersion, buildCommit, buildDate),
	}
	root.AddCommand(
		newInit(),
		newScan(),
		newSnapshot(),
		newApply(),
		newList(),
		newDiff(),
		newKey(),
	)
	return root
}
