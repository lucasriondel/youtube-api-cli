package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newPlaylistsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List the authenticated user's playlists",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			var all []*youtube.Playlist
			pageToken := ""
			for {
				call := svc.Playlists.List([]string{"id", "snippet", "contentDetails", "status"}).
					Mine(true).
					MaxResults(50)
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("playlists.list: %w", err)
				}
				all = append(all, resp.Items...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, p := range all {
				count := ""
				if p.ContentDetails != nil {
					count = fmt.Sprintf("%d", p.ContentDetails.ItemCount)
				}
				privacy := ""
				title := ""
				if p.Snippet != nil {
					title = p.Snippet.Title
				}
				if p.Status != nil {
					privacy = p.Status.PrivacyStatus
				}
				rows = append(rows, []string{p.Id, title, count, privacy})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"ID", "TITLE", "ITEMS", "PRIVACY"},
				rows,
				all,
			)
		},
	}
}
