package main

import (
	"fmt"
	"os"

	"github.com/djleonskennedy/ferry/cmd/ferry/commands"
	"github.com/djleonskennedy/ferry/internal/cliutil"
)

// Build-time vars, set via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func main() {
	commands.SetBuildInfo(Version, Commit, Date)
	if err := commands.NewRoot().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ferry: "+err.Error())
		os.Exit(cliutil.ExitCodeFor(err))
	}
}
