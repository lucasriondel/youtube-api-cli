package cmd

import "github.com/spf13/cobra"

func newChannelsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "channels",
		Short: "Inspect and manage channels",
	}
	c.AddCommand(newChannelsShowCmd())
	c.AddCommand(newChannelsUpdateCmd())
	return c
}
