package cmd

import (
	"context"
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newLikedListCmd() *cobra.Command {
	var noCache bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List videos you've liked",
		Long: "List the videos in your 'Liked videos' playlist (LL).\n\n" +
			"Resolves the LL playlist id via channels.list (mine=true, parts=contentDetails),\n" +
			"then enumerates playlistItems.list with the same etag-aware on-disk cache used\n" +
			"by `items list` — repeated invocations cost 0 quota units when the playlist is\n" +
			"unchanged.\n\n" +
			"Cost: 1 unit (channels.list) + 1 unit per page of playlistItems.list. Pass\n" +
			"--no-cache to bypass the etag cache.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			likesID, err := fetchLikesPlaylistID(ctx, svc)
			if err != nil {
				return err
			}

			parts := []string{"id", "snippet", "contentDetails", "status"}
			var all []*youtube.PlaylistItem
			pageToken := ""
			for {
				resp, err := fetchPlaylistItemsPage(ctx, svc, likesID, parts, pageToken, noCache)
				if err != nil {
					return err
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
	c.Flags().BoolVar(&noCache, "no-cache", false, "skip the on-disk etag cache (always re-fetch)")
	return c
}

// fetchLikesPlaylistID returns the authenticated user's "Liked videos" playlist
// id (the channel-specific LL... id) by reading channels.list contentDetails.
func fetchLikesPlaylistID(ctx context.Context, svc *youtube.Service) (string, error) {
	resp, err := svc.Channels.
		List([]string{"contentDetails"}).
		Mine(true).
		MaxResults(1).
		Context(ctx).
		Do()
	if err != nil {
		return "", fmt.Errorf("channels.list: %w", err)
	}
	if len(resp.Items) == 0 {
		return "", fmt.Errorf("channels.list returned no channel for the authenticated user")
	}
	ch := resp.Items[0]
	if ch.ContentDetails == nil || ch.ContentDetails.RelatedPlaylists == nil || ch.ContentDetails.RelatedPlaylists.Likes == "" {
		return "", fmt.Errorf("channel %s has no relatedPlaylists.likes (Liked videos playlist not exposed)", ch.Id)
	}
	return ch.ContentDetails.RelatedPlaylists.Likes, nil
}
