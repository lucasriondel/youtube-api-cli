package cmd

import "github.com/spf13/cobra"

func newSubsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "subs",
		Short: "Inspect and manage your subscriptions",
	}
	c.AddCommand(newSubsListCmd())
	c.AddCommand(newSubsAddCmd())
	c.AddCommand(newSubsRemoveCmd())
	return c
}
