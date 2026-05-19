package cliutil

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// PrintTable writes header + rows to w using a tabwriter.
func PrintTable(w io.Writer, header []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(header, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	tw.Flush()
}
