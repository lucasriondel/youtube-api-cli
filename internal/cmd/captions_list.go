package cmd

import (
	"fmt"
	"time"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newCaptionsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <video-id-or-url>",
		Short: fmt.Sprintf("List caption tracks for a video (cost: %d units)", ytapi.CostCaptionsList),
		Long: fmt.Sprintf("List caption tracks available on a video via captions.list.\n\n"+
			"Cost: %d units per call (the captions endpoint is unusually expensive\n"+
			"for a read — captions.download is %d units on top of that, so plan\n"+
			"calls accordingly).\n\n"+
			"Accepts a raw 11-char video id or a YouTube URL (watch?v=...,\n"+
			"youtu.be/..., shorts/..., embed/..., v/...).\n\n"+
			"Auto-generated tracks (TrackKind=ASR) and uploader-supplied tracks\n"+
			"(TrackKind=standard) are both returned. The CAPTION_ID column is the\n"+
			"id to feed to `yt captions download <caption-id>`.",
			ytapi.CostCaptionsList, ytapi.CostCaptionsDownload),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseVideoID(args[0])
			if err != nil {
				return fmt.Errorf("invalid video reference %q: %w", args[0], err)
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			resp, err := svc.Captions.List([]string{"id", "snippet"}, id).Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("captions.list: %w", err)
			}

			rows := make([][]string, 0, len(resp.Items))
			for _, c := range resp.Items {
				rows = append(rows, []string{
					c.Id,
					captionLanguage(c),
					captionName(c),
					captionTrackKind(c),
					captionStatus(c),
					captionLastUpdated(c),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"CAPTION_ID", "LANGUAGE", "NAME", "TRACK_KIND", "STATUS", "LAST_UPDATED"},
				rows,
				resp.Items,
			)
		},
	}
}

func captionLanguage(c *youtube.Caption) string {
	if c.Snippet == nil {
		return ""
	}
	return c.Snippet.Language
}

func captionName(c *youtube.Caption) string {
	if c.Snippet == nil {
		return ""
	}
	return c.Snippet.Name
}

func captionTrackKind(c *youtube.Caption) string {
	if c.Snippet == nil {
		return ""
	}
	return c.Snippet.TrackKind
}

func captionStatus(c *youtube.Caption) string {
	if c.Snippet == nil {
		return ""
	}
	return c.Snippet.Status
}

func captionLastUpdated(c *youtube.Caption) string {
	if c.Snippet == nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, c.Snippet.LastUpdated); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return c.Snippet.LastUpdated
}
