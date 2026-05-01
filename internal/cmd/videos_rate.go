package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newVideosRateCmd() *cobra.Command {
	var rating string

	c := &cobra.Command{
		Use:   "rate <id-or-url>",
		Short: fmt.Sprintf("Rate a video (like/dislike/none) (cost: %d units)", ytapi.CostRate),
		Long: fmt.Sprintf("Rate a video as 'like', 'dislike', or remove the rating with 'none'.\n\n"+
			"The argument may be a raw video id (e.g. dQw4w9WgXcQ) or a YouTube URL\n"+
			"(watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Quota cost: %d units per call.\n"+
			"Use --dry-run to print the planned mutation without calling the API.",
			ytapi.CostRate),
		Args: cobra.ExactArgs(1),
	}

	c.Flags().StringVar(&rating, "as", "", "rating to apply: like, dislike, or none (required)")
	_ = c.MarkFlagRequired("as")
	dryRun := addDryRunFlag(c)

	c.RunE = func(cmd *cobra.Command, args []string) error {
		if rating != "like" && rating != "dislike" && rating != "none" {
			return fmt.Errorf("invalid --as %q (want one of: like, dislike, none)", rating)
		}

		videoID, err := parseVideoID(args[0])
		if err != nil {
			return fmt.Errorf("invalid video reference %q: %w", args[0], err)
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostRate,
				"would set rating=%s on video %s", rating, videoID)
			return nil
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		if err := svc.Videos.Rate(videoID, rating).Context(ctx).Do(); err != nil {
			return fmt.Errorf("videos.rate (videoId=%s, rating=%s): %w", videoID, rating, err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Rated video id=%s as=%s\n", videoID, rating)
		return nil
	}

	return c
}
