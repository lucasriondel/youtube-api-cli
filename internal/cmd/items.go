package cmd

import "github.com/spf13/cobra"

func newItemsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "items",
		Short: "Inspect and manage items inside a playlist",
	}
	c.AddCommand(newItemsListCmd())
	c.AddCommand(newItemsAddCmd())
	c.AddCommand(newItemsRemoveCmd())
	c.AddCommand(newItemsMoveCmd())
	c.AddCommand(newItemsSortCmd())
	c.AddCommand(newItemsDedupeCmd())
	return c
}
