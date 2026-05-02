package cmd

import "github.com/spf13/cobra"

func newCommentsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "comments",
		Short: "List comment threads or expand a single thread's replies",
	}
	c.AddCommand(newCommentsListCmd())
	c.AddCommand(newCommentsThreadCmd())
	return c
}
