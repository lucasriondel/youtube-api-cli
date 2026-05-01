package cmd

import (
	"fmt"
	"time"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newSubsListCmd() *cobra.Command {
	var order string
	c := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List the authenticated user's subscriptions (cost: %d unit per page)", ytapi.CostList),
		Long: fmt.Sprintf("List the authenticated user's subscriptions via subscriptions.list (mine=true).\n\n"+
			"Cost: %d unit per page (50 subscriptions per page). Pages are fetched until\n"+
			"exhaustion.\n\n"+
			"--order controls server-side ordering: alphabetical (default), relevance, or\n"+
			"unread.",
			ytapi.CostList),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateSubsOrder(order); err != nil {
				return err
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			var all []*youtube.Subscription
			pageToken := ""
			for {
				call := svc.Subscriptions.List([]string{"id", "snippet", "contentDetails"}).
					Mine(true).
					Order(order).
					MaxResults(50)
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("subscriptions.list: %w", err)
				}
				all = append(all, resp.Items...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, s := range all {
				rows = append(rows, []string{
					s.Id,
					subsChannelID(s),
					subsTitle(s),
					subsItemCount(s),
					subsSubscribedAt(s),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"SUB_ID", "CHANNEL_ID", "TITLE", "ITEMS", "SUBSCRIBED"},
				rows,
				all,
			)
		},
	}
	c.Flags().StringVar(&order, "order", "alphabetical", "result ordering: alphabetical, relevance, unread")
	return c
}

func validateSubsOrder(o string) error {
	switch o {
	case "alphabetical", "relevance", "unread":
		return nil
	default:
		return fmt.Errorf("invalid --order %q (want one of: alphabetical, relevance, unread)", o)
	}
}

func subsChannelID(s *youtube.Subscription) string {
	if s.Snippet == nil || s.Snippet.ResourceId == nil {
		return ""
	}
	return s.Snippet.ResourceId.ChannelId
}

func subsTitle(s *youtube.Subscription) string {
	if s.Snippet == nil {
		return ""
	}
	return s.Snippet.Title
}

func subsItemCount(s *youtube.Subscription) string {
	if s.ContentDetails == nil {
		return ""
	}
	return fmt.Sprintf("%d", s.ContentDetails.TotalItemCount)
}

func subsSubscribedAt(s *youtube.Subscription) string {
	if s.Snippet == nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, s.Snippet.PublishedAt); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return s.Snippet.PublishedAt
}
