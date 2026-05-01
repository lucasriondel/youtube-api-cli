package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newPlaylistsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show a single playlist by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			id := args[0]
			resp, err := svc.Playlists.
				List([]string{"id", "snippet", "contentDetails", "status"}).
				Id(id).
				MaxResults(1).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("playlists.list: %w", err)
			}
			if len(resp.Items) == 0 {
				return fmt.Errorf("playlist %q not found", id)
			}
			p := resp.Items[0]

			title, channel, privacy, description := "", "", "", ""
			if p.Snippet != nil {
				title = p.Snippet.Title
				channel = p.Snippet.ChannelTitle
				description = p.Snippet.Description
			}
			if p.Status != nil {
				privacy = p.Status.PrivacyStatus
			}
			count := ""
			if p.ContentDetails != nil {
				count = fmt.Sprintf("%d", p.ContentDetails.ItemCount)
			}

			rows := [][]string{
				{"ID", p.Id},
				{"TITLE", title},
				{"CHANNEL", channel},
				{"ITEMS", count},
				{"PRIVACY", privacy},
				{"DESCRIPTION", description},
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"FIELD", "VALUE"},
				rows,
				p,
			)
		},
	}
}
