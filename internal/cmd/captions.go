package cmd

import "github.com/spf13/cobra"

func newCaptionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "captions",
		Short: "List and download caption tracks for a video",
	}
	c.AddCommand(newCaptionsListCmd())
	c.AddCommand(newCaptionsDownloadCmd())
	return c
}
