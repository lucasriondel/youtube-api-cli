package cmd

import "github.com/spf13/cobra"

func newCommentsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "comments",
		Short: "List comment threads on videos or channels",
	}
	c.AddCommand(newCommentsListCmd())
	return c
}
