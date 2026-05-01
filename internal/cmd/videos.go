package cmd

import "github.com/spf13/cobra"

func newVideosCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "videos",
		Short: "Inspect and manage videos",
	}
	c.AddCommand(newVideosShowCmd())
	c.AddCommand(newVideosRateCmd())
	c.AddCommand(newVideosRatingCmd())
	c.AddCommand(newVideosUpdateCmd())
	return c
}
