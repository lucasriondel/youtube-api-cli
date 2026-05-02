package cmd

import "github.com/spf13/cobra"

func newCommentsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "comments",
		Short: "List, expand, post, or reply to comment threads",
	}
	c.AddCommand(newCommentsListCmd())
	c.AddCommand(newCommentsThreadCmd())
	c.AddCommand(newCommentsPostCmd())
	c.AddCommand(newCommentsReplyCmd())
	return c
}
