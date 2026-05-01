package cmd

import "github.com/spf13/cobra"

func newLikedCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "liked",
		Short: "Inspect your liked videos",
	}
	c.AddCommand(newLikedListCmd())
	return c
}
