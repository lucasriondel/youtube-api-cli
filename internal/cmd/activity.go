package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newActivityCmd() *cobra.Command {
	var (
		mine      bool
		channelID string
		since     string
		max       int64
	)

	c := &cobra.Command{
		Use:   "activity",
		Short: fmt.Sprintf("List recent channel activities (cost: %d unit per page)", ytapi.CostList),
		Long: fmt.Sprintf("List recent activities (uploads, likes, subscriptions, playlist additions, etc.) "+
			"via activities.list.\n\n"+
			"Cost: %d unit per page (50 activities per page).\n\n"+
			"Exactly one of --mine or --channel <id> must be provided. --since accepts an\n"+
			"RFC3339 timestamp (e.g. 2026-01-01T00:00:00Z) or a YYYY-MM-DD date (interpreted\n"+
			"as UTC midnight) and is passed to the API as publishedAfter.\n\n"+
			"--max caps the total number of activities returned across pages. The API\n"+
			"returns up to 50 per page, so --max>50 forces additional %d-unit calls.\n\n"+
			"Note: the activities.list API has known limitations — it only surfaces a\n"+
			"subset of activity types and may return fewer items than the channel actually\n"+
			"produced. Use it as a recent-changes feed, not an audit log.",
			ytapi.CostList, ytapi.CostList),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mine && channelID != "" {
				return fmt.Errorf("--mine is mutually exclusive with --channel")
			}
			if !mine && channelID == "" {
				return fmt.Errorf("provide either --mine or --channel <id>")
			}
			if max <= 0 {
				return fmt.Errorf("--max must be > 0")
			}

			publishedAfter := ""
			if since != "" {
				ts, err := parseSince(since)
				if err != nil {
					return fmt.Errorf("invalid --since %q: %w", since, err)
				}
				publishedAfter = ts
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			var all []*youtube.Activity
			pageToken := ""
			remaining := max
			for remaining > 0 {
				thisPage := remaining
				if thisPage > 50 {
					thisPage = 50
				}
				call := svc.Activities.List([]string{"id", "snippet", "contentDetails"}).
					MaxResults(thisPage)
				if mine {
					call = call.Mine(true)
				} else {
					call = call.ChannelId(channelID)
				}
				if publishedAfter != "" {
					call = call.PublishedAfter(publishedAfter)
				}
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("activities.list: %w", err)
				}
				all = append(all, resp.Items...)
				remaining -= int64(len(resp.Items))
				if resp.NextPageToken == "" || len(resp.Items) == 0 {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, a := range all {
				rows = append(rows, []string{
					activityPublished(a),
					activityType(a),
					activityResourceID(a),
					activityTitle(a),
					activityChannel(a),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"PUBLISHED", "TYPE", "RESOURCE_ID", "TITLE", "CHANNEL"},
				rows,
				all,
			)
		},
	}

	c.Flags().BoolVar(&mine, "mine", false, "list activities for the authenticated user's own channel")
	c.Flags().StringVar(&channelID, "channel", "", "list activities for the given channel id (UC...)")
	c.Flags().StringVar(&since, "since", "", "only return activities published after this date (RFC3339 or YYYY-MM-DD)")
	c.Flags().Int64Var(&max, "max", 50, "maximum number of activities to return across pages")
	return c
}

func parseSince(s string) (string, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC().Format(time.RFC3339), nil
	}
	return "", fmt.Errorf("expected RFC3339 timestamp or YYYY-MM-DD date")
}

func activityPublished(a *youtube.Activity) string {
	if a.Snippet == nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, a.Snippet.PublishedAt); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return a.Snippet.PublishedAt
}

func activityType(a *youtube.Activity) string {
	if a.Snippet == nil {
		return ""
	}
	return a.Snippet.Type
}

func activityChannel(a *youtube.Activity) string {
	if a.Snippet == nil {
		return ""
	}
	return a.Snippet.ChannelTitle
}

func activityTitle(a *youtube.Activity) string {
	if a.Snippet == nil {
		return ""
	}
	return strings.TrimSpace(a.Snippet.Title)
}

func activityResourceID(a *youtube.Activity) string {
	if a.ContentDetails == nil {
		return ""
	}
	cd := a.ContentDetails
	switch {
	case cd.Upload != nil && cd.Upload.VideoId != "":
		return cd.Upload.VideoId
	case cd.Like != nil && cd.Like.ResourceId != nil && cd.Like.ResourceId.VideoId != "":
		return cd.Like.ResourceId.VideoId
	case cd.Favorite != nil && cd.Favorite.ResourceId != nil && cd.Favorite.ResourceId.VideoId != "":
		return cd.Favorite.ResourceId.VideoId
	case cd.Subscription != nil && cd.Subscription.ResourceId != nil && cd.Subscription.ResourceId.ChannelId != "":
		return cd.Subscription.ResourceId.ChannelId
	case cd.PlaylistItem != nil && cd.PlaylistItem.ResourceId != nil && cd.PlaylistItem.ResourceId.VideoId != "":
		return cd.PlaylistItem.ResourceId.VideoId
	case cd.Bulletin != nil && cd.Bulletin.ResourceId != nil:
		rid := cd.Bulletin.ResourceId
		if rid.VideoId != "" {
			return rid.VideoId
		}
		if rid.PlaylistId != "" {
			return rid.PlaylistId
		}
		if rid.ChannelId != "" {
			return rid.ChannelId
		}
	case cd.ChannelItem != nil && cd.ChannelItem.ResourceId != nil:
		rid := cd.ChannelItem.ResourceId
		if rid.VideoId != "" {
			return rid.VideoId
		}
		if rid.ChannelId != "" {
			return rid.ChannelId
		}
	case cd.Recommendation != nil && cd.Recommendation.ResourceId != nil:
		rid := cd.Recommendation.ResourceId
		if rid.VideoId != "" {
			return rid.VideoId
		}
		if rid.ChannelId != "" {
			return rid.ChannelId
		}
	case cd.Social != nil && cd.Social.ResourceId != nil:
		rid := cd.Social.ResourceId
		if rid.VideoId != "" {
			return rid.VideoId
		}
		if rid.ChannelId != "" {
			return rid.ChannelId
		}
	case cd.Comment != nil && cd.Comment.ResourceId != nil:
		rid := cd.Comment.ResourceId
		if rid.VideoId != "" {
			return rid.VideoId
		}
		if rid.ChannelId != "" {
			return rid.ChannelId
		}
	}
	return ""
}
