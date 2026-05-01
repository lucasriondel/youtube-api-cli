package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newItemsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <playlist-id> <video-id-or-url>...",
		Short: fmt.Sprintf("Add one or more videos to a playlist (cost: %d units per video)", ytapi.CostInsert),
		Long: fmt.Sprintf("Add videos to a playlist. Each argument may be a raw video id (e.g. dQw4w9WgXcQ)\n"+
			"or a YouTube URL (watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Quota cost: %d units per video added.\n"+
			"Use --dry-run to print the planned mutations without calling the API.", ytapi.CostInsert),
		Args: cobra.MinimumNArgs(2),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		playlistID := args[0]
		rawVideos := args[1:]

		videoIDs := make([]string, 0, len(rawVideos))
		for _, raw := range rawVideos {
			id, err := parseVideoID(raw)
			if err != nil {
				return fmt.Errorf("invalid video reference %q: %w", raw, err)
			}
			videoIDs = append(videoIDs, id)
		}

		totalCost := ytapi.CostInsert * len(videoIDs)

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), totalCost,
				"would add %d video(s) to playlist %s",
				len(videoIDs), playlistID,
			)
			rows := make([][]string, 0, len(videoIDs))
			for _, vid := range videoIDs {
				rows = append(rows, []string{playlistID, vid})
			}
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"PLAYLIST_ID", "VIDEO_ID"},
				rows,
				videoIDs,
			)
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		inserted := make([]*youtube.PlaylistItem, 0, len(videoIDs))
		rows := make([][]string, 0, len(videoIDs))
		for _, vid := range videoIDs {
			item := &youtube.PlaylistItem{
				Snippet: &youtube.PlaylistItemSnippet{
					PlaylistId: playlistID,
					ResourceId: &youtube.ResourceId{
						Kind:    "youtube#video",
						VideoId: vid,
					},
				},
			}
			created, err := svc.PlaylistItems.
				Insert([]string{"snippet"}, item).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("playlistItems.insert (videoId=%s): %w", vid, err)
			}
			inserted = append(inserted, created)

			position, title := "", ""
			if created.Snippet != nil {
				position = fmt.Sprintf("%d", created.Snippet.Position)
				title = created.Snippet.Title
			}
			rows = append(rows, []string{position, created.Id, vid, title})
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"POS", "ITEM_ID", "VIDEO_ID", "TITLE"},
			rows,
			inserted,
		)
	}

	return c
}

// parseVideoID accepts either a raw 11-char video id or a YouTube URL and
// returns the canonical video id. Supports watch?v=, youtu.be/, /shorts/,
// /embed/, and /v/ paths.
func parseVideoID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty value")
	}
	if !strings.Contains(s, "/") && !strings.Contains(s, "?") {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("not a valid id or URL: %w", err)
	}
	if v := u.Query().Get("v"); v != "" {
		return v, nil
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch {
	case u.Host == "youtu.be" && len(parts) >= 1 && parts[0] != "":
		return parts[0], nil
	case len(parts) >= 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "v"):
		return parts[1], nil
	}
	return "", fmt.Errorf("could not extract video id from URL")
}
