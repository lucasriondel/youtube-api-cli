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

func newSearchCmd() *cobra.Command {
	var (
		searchType string
		max        int64
		channelID  string
		order      string
	)

	c := &cobra.Command{
		Use:   "search <query>",
		Short: fmt.Sprintf("Search YouTube for videos, channels, or playlists (cost: %d units per call)", ytapi.CostSearchList),
		Long: fmt.Sprintf("Search YouTube via search.list.\n\n"+
			"Quota cost: %d units per call. This is the most expensive read in the API —\n"+
			"a single page (up to 50 results) costs %d times a normal list read. The default\n"+
			"daily quota of 10,000 units is %d searches.\n\n"+
			"--max caps the total number of results. The API returns up to 50 per page, so\n"+
			"--max>50 forces additional %d-unit calls; the command warns on stderr before\n"+
			"spending the extra quota.\n\n"+
			"--type filters by resource kind (video, channel, playlist, or any). Default 'any'\n"+
			"returns a mix; the TYPE column tells them apart.\n\n"+
			"--channel scopes the search to a single channel id (search.list channelId param).\n"+
			"--order controls ranking: relevance (default), date, rating, viewCount, title.",
			ytapi.CostSearchList, ytapi.CostSearchList,
			10000/ytapi.CostSearchList, ytapi.CostSearchList),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.TrimSpace(args[0])
			if query == "" {
				return fmt.Errorf("empty search query")
			}
			if err := validateSearchType(searchType); err != nil {
				return err
			}
			if err := validateSearchOrder(order); err != nil {
				return err
			}
			if max <= 0 {
				return fmt.Errorf("--max must be > 0")
			}

			pageSize := max
			if pageSize > 50 {
				pageSize = 50
			}
			estimatedPages := (max + 49) / 50
			if estimatedPages > 1 {
				fmt.Fprintf(cmd.OutOrStderr(),
					"warning: --max=%d will require up to %d calls = up to %d quota units (search.list is %d units/call)\n",
					max, estimatedPages, estimatedPages*ytapi.CostSearchList, ytapi.CostSearchList,
				)
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			var all []*youtube.SearchResult
			pageToken := ""
			remaining := max
			for remaining > 0 {
				thisPage := remaining
				if thisPage > 50 {
					thisPage = 50
				}
				call := svc.Search.List([]string{"id", "snippet"}).
					Q(query).
					MaxResults(thisPage)
				if searchType != "any" {
					call = call.Type(searchType)
				}
				if channelID != "" {
					call = call.ChannelId(channelID)
				}
				if order != "" {
					call = call.Order(order)
				}
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("search.list: %w", err)
				}
				all = append(all, resp.Items...)
				remaining -= int64(len(resp.Items))
				if resp.NextPageToken == "" || len(resp.Items) == 0 {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, r := range all {
				rows = append(rows, []string{
					searchResultType(r),
					searchResultID(r),
					searchResultTitle(r),
					searchResultChannel(r),
					searchResultPublished(r),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"TYPE", "ID", "TITLE", "CHANNEL", "PUBLISHED"},
				rows,
				all,
			)
		},
	}

	c.Flags().StringVar(&searchType, "type", "any", "resource type to return: video, channel, playlist, or any")
	c.Flags().Int64Var(&max, "max", 25, "maximum number of results to return (1-500). >50 forces additional 100-unit calls.")
	c.Flags().StringVar(&channelID, "channel", "", "restrict search to a single channel id")
	c.Flags().StringVar(&order, "order", "relevance", "result ordering: relevance, date, rating, viewCount, title")
	return c
}

func validateSearchType(t string) error {
	switch t {
	case "any", "video", "channel", "playlist":
		return nil
	default:
		return fmt.Errorf("invalid --type %q (want one of: any, video, channel, playlist)", t)
	}
}

func validateSearchOrder(o string) error {
	switch o {
	case "relevance", "date", "rating", "viewCount", "title":
		return nil
	default:
		return fmt.Errorf("invalid --order %q (want one of: relevance, date, rating, viewCount, title)", o)
	}
}

func searchResultType(r *youtube.SearchResult) string {
	if r.Id == nil {
		return ""
	}
	switch {
	case r.Id.VideoId != "":
		return "video"
	case r.Id.ChannelId != "" && r.Id.Kind == "youtube#channel":
		return "channel"
	case r.Id.PlaylistId != "":
		return "playlist"
	}
	return strings.TrimPrefix(r.Id.Kind, "youtube#")
}

func searchResultID(r *youtube.SearchResult) string {
	if r.Id == nil {
		return ""
	}
	switch {
	case r.Id.VideoId != "":
		return r.Id.VideoId
	case r.Id.PlaylistId != "":
		return r.Id.PlaylistId
	case r.Id.ChannelId != "":
		return r.Id.ChannelId
	}
	return ""
}

func searchResultTitle(r *youtube.SearchResult) string {
	if r.Snippet == nil {
		return ""
	}
	return r.Snippet.Title
}

func searchResultChannel(r *youtube.SearchResult) string {
	if r.Snippet == nil {
		return ""
	}
	return r.Snippet.ChannelTitle
}

func searchResultPublished(r *youtube.SearchResult) string {
	if r.Snippet == nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, r.Snippet.PublishedAt); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return r.Snippet.PublishedAt
}
