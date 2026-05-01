package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newItemsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <playlist-id>",
		Short: "List the items in a playlist",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			playlistID := args[0]

			var all []*youtube.PlaylistItem
			pageToken := ""
			for {
				call := svc.PlaylistItems.
					List([]string{"id", "snippet", "contentDetails", "status"}).
					PlaylistId(playlistID).
					MaxResults(50)
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("playlistItems.list: %w", err)
				}
				all = append(all, resp.Items...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, it := range all {
				position, videoID, title, channel := "", "", "", ""
				if it.Snippet != nil {
					position = fmt.Sprintf("%d", it.Snippet.Position)
					title = it.Snippet.Title
					channel = it.Snippet.VideoOwnerChannelTitle
					if it.Snippet.ResourceId != nil {
						videoID = it.Snippet.ResourceId.VideoId
					}
				}
				if videoID == "" && it.ContentDetails != nil {
					videoID = it.ContentDetails.VideoId
				}
				rows = append(rows, []string{position, it.Id, videoID, title, channel})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"POS", "ITEM_ID", "VIDEO_ID", "TITLE", "CHANNEL"},
				rows,
				all,
			)
		},
	}
}
