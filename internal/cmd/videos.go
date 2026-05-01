package cmd

import "github.com/spf13/cobra"

func newVideosCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "videos",
		Short: "Inspect and manage videos",
	}
	c.AddCommand(newVideosShowCmd())
	return c
}
