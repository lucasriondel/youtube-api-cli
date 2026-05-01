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

func newChannelsShowCmd() *cobra.Command {
	var mine bool
	c := &cobra.Command{
		Use:   "show [<id>...]",
		Short: fmt.Sprintf("Show metadata for one or more channels (cost: %d unit per call)", ytapi.CostList),
		Long: fmt.Sprintf("Show metadata for one or more channels via channels.list.\n\n"+
			"Each argument is a channel id (e.g. UCXMq1fqwkAD_Ol0HeInZrwg). Pass --mine to\n"+
			"show the authenticated user's own channel instead — this is mutually exclusive\n"+
			"with positional ids.\n\n"+
			"Fetches snippet,contentDetails,statistics,brandingSettings.\n\n"+
			"Quota cost: %d unit per call (channels.list accepts up to 50 ids per call).",
			ytapi.CostList),
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mine && len(args) > 0 {
				return fmt.Errorf("--mine is mutually exclusive with channel id arguments")
			}
			if !mine && len(args) == 0 {
				return fmt.Errorf("provide at least one channel id, or pass --mine")
			}
			for _, raw := range args {
				if strings.TrimSpace(raw) == "" {
					return fmt.Errorf("invalid channel id %q: empty value", raw)
				}
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			parts := []string{"id", "snippet", "contentDetails", "statistics", "brandingSettings"}
			var all []*youtube.Channel
			if mine {
				resp, err := svc.Channels.
					List(parts).
					Mine(true).
					MaxResults(50).
					Context(ctx).
					Do()
				if err != nil {
					return fmt.Errorf("channels.list: %w", err)
				}
				all = resp.Items
			} else {
				for i := 0; i < len(args); i += 50 {
					end := i + 50
					if end > len(args) {
						end = len(args)
					}
					batch := args[i:end]
					resp, err := svc.Channels.
						List(parts).
						Id(strings.Join(batch, ",")).
						MaxResults(50).
						Context(ctx).
						Do()
					if err != nil {
						return fmt.Errorf("channels.list: %w", err)
					}
					all = append(all, resp.Items...)
				}

				returned := make(map[string]bool, len(all))
				for _, ch := range all {
					returned[ch.Id] = true
				}
				missing := make([]string, 0)
				for _, id := range args {
					if !returned[id] {
						missing = append(missing, id)
					}
				}
				if len(missing) > 0 {
					fmt.Fprintf(cmd.OutOrStderr(),
						"warning: %d channel(s) not found or unavailable: %s\n",
						len(missing), strings.Join(missing, ", "),
					)
				}
			}

			rows := make([][]string, 0, len(all))
			for _, ch := range all {
				rows = append(rows, []string{
					ch.Id,
					channelTitle(ch),
					channelHandle(ch),
					channelSubs(ch),
					channelVideos(ch),
					channelViews(ch),
					channelPublished(ch),
					channelCountry(ch),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"ID", "TITLE", "HANDLE", "SUBS", "VIDEOS", "VIEWS", "PUBLISHED", "COUNTRY"},
				rows,
				all,
			)
		},
	}
	c.Flags().BoolVar(&mine, "mine", false, "show the authenticated user's own channel")
	return c
}

func channelTitle(ch *youtube.Channel) string {
	if ch.Snippet == nil {
		return ""
	}
	return ch.Snippet.Title
}

func channelHandle(ch *youtube.Channel) string {
	if ch.Snippet == nil {
		return ""
	}
	return ch.Snippet.CustomUrl
}

func channelSubs(ch *youtube.Channel) string {
	if ch.Statistics == nil {
		return ""
	}
	if ch.Statistics.HiddenSubscriberCount {
		return "hidden"
	}
	return fmt.Sprintf("%d", ch.Statistics.SubscriberCount)
}

func channelVideos(ch *youtube.Channel) string {
	if ch.Statistics == nil {
		return ""
	}
	return fmt.Sprintf("%d", ch.Statistics.VideoCount)
}

func channelViews(ch *youtube.Channel) string {
	if ch.Statistics == nil {
		return ""
	}
	return fmt.Sprintf("%d", ch.Statistics.ViewCount)
}

func channelPublished(ch *youtube.Channel) string {
	if ch.Snippet == nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, ch.Snippet.PublishedAt); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return ch.Snippet.PublishedAt
}

func channelCountry(ch *youtube.Channel) string {
	if ch.Snippet == nil {
		return ""
	}
	return ch.Snippet.Country
}
