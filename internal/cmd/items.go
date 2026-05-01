package cmd

import "github.com/spf13/cobra"

func newItemsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "items",
		Short: "Inspect and manage items inside a playlist",
	}
	c.AddCommand(newItemsListCmd())
	return c
}
