package cmd

import "github.com/spf13/cobra"

func newLikedCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "liked",
		Short: "Inspect or change your liked videos",
	}
	c.AddCommand(newLikedListCmd())
	c.AddCommand(newLikedAddCmd())
	c.AddCommand(newLikedRemoveCmd())
	return c
}
