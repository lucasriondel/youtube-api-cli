package cmd

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// addDryRunFlag registers the standard --dry-run flag against cmd and returns
// a pointer to the bound bool. Every write command should use this so the flag
// description stays uniform.
func addDryRunFlag(cmd *cobra.Command) *bool {
	dryRun := false
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the planned mutation(s) and quota cost without calling the API")
	return &dryRun
}

// printDryRun writes a uniform "DRY RUN: <action> (cost: N units)" line to w.
// Use this from every write command's --dry-run branch so the format is
// consistent for agents grepping output.
func printDryRun(w io.Writer, costUnits int, format string, args ...any) {
	action := fmt.Sprintf(format, args...)
	fmt.Fprintf(w, "DRY RUN: %s (cost: %d units)\n", action, costUnits)
}
