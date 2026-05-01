package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newSubsAddCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add <channel-id>...",
		Short: fmt.Sprintf("Subscribe to one or more channels (cost: %d units per channel)", ytapi.CostInsert),
		Long: fmt.Sprintf("Subscribe the authenticated user to each channel via subscriptions.insert.\n\n"+
			"Each argument is a channel id (typically a 24-char string starting with UC).\n"+
			"Channel handles (@name) and URLs are not supported here — resolve them first\n"+
			"with `yt search` or `yt channels show`.\n\n"+
			"Quota cost: %d units per channel subscribed.\n"+
			"Use --dry-run to print the planned mutations without calling the API.",
			ytapi.CostInsert),
		Args: cobra.MinimumNArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		channelIDs := make([]string, 0, len(args))
		for _, raw := range args {
			id := strings.TrimSpace(raw)
			if id == "" {
				return fmt.Errorf("invalid channel id %q: empty value", raw)
			}
			channelIDs = append(channelIDs, id)
		}

		totalCost := ytapi.CostInsert * len(channelIDs)

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), totalCost,
				"would subscribe to %d channel(s)",
				len(channelIDs),
			)
			rows := make([][]string, 0, len(channelIDs))
			for _, cid := range channelIDs {
				rows = append(rows, []string{cid})
			}
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"CHANNEL_ID"},
				rows,
				channelIDs,
			)
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		created := make([]*youtube.Subscription, 0, len(channelIDs))
		rows := make([][]string, 0, len(channelIDs))
		for _, cid := range channelIDs {
			sub := &youtube.Subscription{
				Snippet: &youtube.SubscriptionSnippet{
					ResourceId: &youtube.ResourceId{
						Kind:      "youtube#channel",
						ChannelId: cid,
					},
				},
			}
			resp, err := svc.Subscriptions.
				Insert([]string{"snippet"}, sub).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("subscriptions.insert (channelId=%s): %w", cid, err)
			}
			created = append(created, resp)

			title := ""
			if resp.Snippet != nil {
				title = resp.Snippet.Title
			}
			rows = append(rows, []string{resp.Id, cid, title})
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"SUB_ID", "CHANNEL_ID", "TITLE"},
			rows,
			created,
		)
	}

	return c
}
