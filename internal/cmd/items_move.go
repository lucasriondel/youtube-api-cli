package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newItemsMoveCmd() *cobra.Command {
	var (
		to     int64
		dryRun bool
	)

	c := &cobra.Command{
		Use:   "move <playlist-id> <item-id>",
		Short: "Move an item to a new position within its playlist (cost: 51 units)",
		Long: "Move a playlist item to a new zero-based position. The item is identified\n" +
			"by its playlistItemId (the ITEM_ID column from `yt items list`), NOT the videoId.\n\n" +
			"Quota cost: 1 unit (read to look up the item's resourceId) + 50 units (update) = 51 units.\n" +
			"Use --dry-run to print the planned mutation without calling the API.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			playlistID := strings.TrimSpace(args[0])
			itemID := strings.TrimSpace(args[1])
			if playlistID == "" {
				return fmt.Errorf("empty playlist id")
			}
			if itemID == "" {
				return fmt.Errorf("empty item id")
			}
			if !cmd.Flags().Changed("to") {
				return fmt.Errorf("--to <position> is required")
			}
			if to < 0 {
				return fmt.Errorf("--to must be >= 0, got %d", to)
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			resp, err := svc.PlaylistItems.
				List([]string{"id", "snippet"}).
				Id(itemID).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("playlistItems.list (id=%s): %w", itemID, err)
			}
			if len(resp.Items) == 0 {
				return fmt.Errorf("no item found with id=%s", itemID)
			}
			current := resp.Items[0]
			if current.Snippet == nil || current.Snippet.ResourceId == nil {
				return fmt.Errorf("item %s missing snippet/resourceId — cannot update", itemID)
			}
			if current.Snippet.PlaylistId != playlistID {
				return fmt.Errorf("item %s is in playlist %s, not %s", itemID, current.Snippet.PlaylistId, playlistID)
			}

			if dryRun {
				fmt.Fprintf(cmd.OutOrStderr(),
					"DRY RUN: would move item %s from position %d to %d in playlist %s (cost: 51 units)\n",
					itemID, current.Snippet.Position, to, playlistID,
				)
				return nil
			}

			updated := &youtube.PlaylistItem{
				Id: itemID,
				Snippet: &youtube.PlaylistItemSnippet{
					PlaylistId: playlistID,
					ResourceId: current.Snippet.ResourceId,
					Position:   to,
				},
			}
			if to == 0 {
				updated.Snippet.ForceSendFields = []string{"Position"}
			}

			result, err := svc.PlaylistItems.
				Update([]string{"snippet"}, updated).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("playlistItems.update (itemId=%s): %w", itemID, err)
			}

			pos, videoID, title := "", "", ""
			if result.Snippet != nil {
				pos = fmt.Sprintf("%d", result.Snippet.Position)
				title = result.Snippet.Title
				if result.Snippet.ResourceId != nil {
					videoID = result.Snippet.ResourceId.VideoId
				}
			}
			rows := [][]string{{pos, result.Id, videoID, title}}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"POS", "ITEM_ID", "VIDEO_ID", "TITLE"},
				rows,
				result,
			)
		},
	}

	c.Flags().Int64Var(&to, "to", 0, "target zero-based position (required)")
	c.Flags().BoolVar(&dryRun, "dry-run", false, "print the planned mutation without calling the API")
	return c
}
