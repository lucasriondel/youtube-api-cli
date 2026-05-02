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

func newCommentsListCmd() *cobra.Command {
	var (
		videoID   string
		channelID string
		order     string
		max       int64
	)

	c := &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List top-level comment threads on a video or channel (cost: %d unit per page)", ytapi.CostList),
		Long: fmt.Sprintf("List top-level comment threads via commentThreads.list.\n\n"+
			"Cost: %d unit per page (up to 100 threads per page). Pages are fetched\n"+
			"until --max is reached or the API runs out of results.\n\n"+
			"Exactly one of --video <id> or --channel <id> must be provided. --video\n"+
			"lists threads on a single video; --channel lists threads on a channel\n"+
			"itself (not threads on the channel's videos — that would require\n"+
			"--all-related, which is not exposed here).\n\n"+
			"--order accepts time (default — newest first) or relevance.\n\n"+
			"Replies are not fetched: only the top-level comment of each thread is\n"+
			"returned. The TotalReplyCount column shows how many replies each thread\n"+
			"has so callers can pick which ones to expand later.",
			ytapi.CostList),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if videoID != "" && channelID != "" {
				return fmt.Errorf("--video is mutually exclusive with --channel")
			}
			if videoID == "" && channelID == "" {
				return fmt.Errorf("provide either --video <id> or --channel <id>")
			}
			if max <= 0 {
				return fmt.Errorf("--max must be > 0")
			}
			if err := validateCommentsOrder(order); err != nil {
				return err
			}

			if videoID != "" {
				id, err := parseVideoID(videoID)
				if err != nil {
					return fmt.Errorf("invalid --video %q: %w", videoID, err)
				}
				videoID = id
			}
			if channelID != "" {
				channelID = strings.TrimSpace(channelID)
				if channelID == "" {
					return fmt.Errorf("invalid --channel: empty value")
				}
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			var all []*youtube.CommentThread
			pageToken := ""
			remaining := max
			for remaining > 0 {
				thisPage := remaining
				if thisPage > 100 {
					thisPage = 100
				}
				call := svc.CommentThreads.List([]string{"id", "snippet"}).
					Order(order).
					TextFormat("plainText").
					MaxResults(thisPage)
				if videoID != "" {
					call = call.VideoId(videoID)
				} else {
					call = call.ChannelId(channelID)
				}
				if pageToken != "" {
					call = call.PageToken(pageToken)
				}
				resp, err := call.Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("commentThreads.list: %w", err)
				}
				all = append(all, resp.Items...)
				remaining -= int64(len(resp.Items))
				if resp.NextPageToken == "" || len(resp.Items) == 0 {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, t := range all {
				rows = append(rows, []string{
					t.Id,
					commentThreadAuthor(t),
					commentThreadPublished(t),
					commentThreadReplyCount(t),
					commentThreadLikes(t),
					commentThreadText(t),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"THREAD_ID", "AUTHOR", "PUBLISHED", "REPLIES", "LIKES", "TEXT"},
				rows,
				all,
			)
		},
	}

	c.Flags().StringVar(&videoID, "video", "", "list threads on this video (raw id or YouTube URL)")
	c.Flags().StringVar(&channelID, "channel", "", "list threads on this channel (raw channel id, UC...)")
	c.Flags().StringVar(&order, "order", "time", "result ordering: time (newest first) or relevance")
	c.Flags().Int64Var(&max, "max", 100, "maximum number of threads to return across pages")
	return c
}

func validateCommentsOrder(o string) error {
	switch o {
	case "time", "relevance":
		return nil
	default:
		return fmt.Errorf("invalid --order %q (want one of: time, relevance)", o)
	}
}

func commentThreadTopLevel(t *youtube.CommentThread) *youtube.CommentSnippet {
	if t.Snippet == nil || t.Snippet.TopLevelComment == nil || t.Snippet.TopLevelComment.Snippet == nil {
		return nil
	}
	return t.Snippet.TopLevelComment.Snippet
}

func commentThreadAuthor(t *youtube.CommentThread) string {
	s := commentThreadTopLevel(t)
	if s == nil {
		return ""
	}
	return s.AuthorDisplayName
}

func commentThreadPublished(t *youtube.CommentThread) string {
	s := commentThreadTopLevel(t)
	if s == nil {
		return ""
	}
	if ts, err := time.Parse(time.RFC3339, s.PublishedAt); err == nil {
		return ts.UTC().Format("2006-01-02")
	}
	return s.PublishedAt
}

func commentThreadReplyCount(t *youtube.CommentThread) string {
	if t.Snippet == nil {
		return "0"
	}
	return fmt.Sprintf("%d", t.Snippet.TotalReplyCount)
}

func commentThreadLikes(t *youtube.CommentThread) string {
	s := commentThreadTopLevel(t)
	if s == nil {
		return "0"
	}
	return fmt.Sprintf("%d", s.LikeCount)
}

func commentThreadText(t *youtube.CommentThread) string {
	s := commentThreadTopLevel(t)
	if s == nil {
		return ""
	}
	return collapseWhitespace(s.TextDisplay)
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
