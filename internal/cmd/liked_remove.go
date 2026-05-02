package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newLikedRemoveCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "remove <id-or-url>...",
		Short: fmt.Sprintf("Unlike one or more videos (cost: %d units per video)", ytapi.CostRate),
		Long: fmt.Sprintf("Remove your rating from one or more videos. Implemented as videos.rate\n"+
			"(rating=none) rather than playlistItems.delete on the LL playlist — Google's\n"+
			"API only supports removing from LL via the rating endpoint.\n\n"+
			"Note: this clears any rating, including 'dislike'. There is no API to scope\n"+
			"the change to like-only.\n\n"+
			"Each argument may be a raw video id (e.g. dQw4w9WgXcQ) or a YouTube URL\n"+
			"(watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Quota cost: %d units per video.\n"+
			"Use --dry-run to print the planned mutations without calling the API.",
			ytapi.CostRate),
		Args: cobra.MinimumNArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		return runLikedRate(cmd, args, "none", *dryRun)
	}

	return c
}

// runLikedRate is the shared implementation behind `liked add` (rating=like)
// and `liked remove` (rating=none). It parses every argument up-front so a bad
// id fails before any API call.
func runLikedRate(cmd *cobra.Command, args []string, rating string, dryRun bool) error {
	videoIDs := make([]string, 0, len(args))
	for _, raw := range args {
		id, err := parseVideoID(raw)
		if err != nil {
			return fmt.Errorf("invalid video reference %q: %w", raw, err)
		}
		videoIDs = append(videoIDs, id)
	}

	totalCost := ytapi.CostRate * len(videoIDs)
	verb := "like"
	if rating == "none" {
		verb = "unlike"
	}

	if dryRun {
		printDryRun(cmd.OutOrStderr(), totalCost,
			"would %s %d video(s)", verb, len(videoIDs))
		rows := make([][]string, 0, len(videoIDs))
		for _, vid := range videoIDs {
			rows = append(rows, []string{vid, rating})
		}
		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"VIDEO_ID", "RATING"},
			rows,
			videoIDs,
		)
	}

	ctx := cmd.Context()
	svc, err := ytapi.New(ctx)
	if err != nil {
		return err
	}

	rows := make([][]string, 0, len(videoIDs))
	for _, vid := range videoIDs {
		if err := svc.Videos.Rate(vid, rating).Context(ctx).Do(); err != nil {
			return fmt.Errorf("videos.rate (videoId=%s, rating=%s): %w", vid, rating, err)
		}
		rows = append(rows, []string{vid, rating})
	}

	format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
	return output.Render(
		cmd.OutOrStdout(),
		format,
		[]string{"VIDEO_ID", "RATING"},
		rows,
		videoIDs,
	)
}
