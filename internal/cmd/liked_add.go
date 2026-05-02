package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newLikedAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <id-or-url>...",
		Short: fmt.Sprintf("Like one or more videos (cost: %d units per video)", ytapi.CostRate),
		Long: fmt.Sprintf("Like one or more videos. Implemented as videos.rate (rating=like)\n"+
			"rather than playlistItems.insert on the LL playlist — Google's API only\n"+
			"supports adding to LL via the rating endpoint.\n\n"+
			"Each argument may be a raw video id (e.g. dQw4w9WgXcQ) or a YouTube URL\n"+
			"(watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Quota cost: %d units per video.\n"+
			"Use --dry-run to print the planned mutations without calling the API.",
			ytapi.CostRate),
		Args: cobra.MinimumNArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return runLikedRate(cmd, args, "like", *dryRun)
	}

	return c
}
