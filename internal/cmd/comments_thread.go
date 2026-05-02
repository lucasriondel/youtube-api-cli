package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newCommentsThreadCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "thread <thread-id>",
		Short: fmt.Sprintf("Show a comment thread with all its replies (cost: %d unit + 1 per extra reply page)", ytapi.CostList),
		Long: fmt.Sprintf("Fetch a single comment thread by id and expand its full reply chain.\n\n"+
			"Cost: %d unit for the commentThreads.list call (returns the top-level\n"+
			"comment plus up to 5 inline replies). If the thread has more replies\n"+
			"than were inlined, an additional comments.list call is paginated\n"+
			"(%d unit per page, up to 100 replies per page) until every reply has\n"+
			"been fetched.\n\n"+
			"Output renders the top-level comment first, then each reply prefixed\n"+
			"with an arrow. JSON output is the raw youtube#commentThread object\n"+
			"with replies.comments populated with the full set of replies (merged\n"+
			"from both calls if pagination was needed).",
			ytapi.CostList, ytapi.CostList),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			threadID := strings.TrimSpace(args[0])
			if threadID == "" {
				return fmt.Errorf("invalid thread id: empty value")
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			resp, err := svc.CommentThreads.List([]string{"id", "snippet", "replies"}).
				Id(threadID).
				TextFormat("plainText").
				MaxResults(1).
				Context(ctx).Do()
			if err != nil {
				return fmt.Errorf("commentThreads.list: %w", err)
			}
			if len(resp.Items) == 0 {
				return fmt.Errorf("thread %q not found (or not accessible to the authenticated account)", threadID)
			}
			thread := resp.Items[0]

			if err := expandReplies(ctx, svc, thread); err != nil {
				return err
			}

			rows := buildThreadRows(thread)
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"COMMENT_ID", "AUTHOR", "PUBLISHED", "LIKES", "TEXT"},
				rows,
				thread,
			)
		},
	}
	return c
}

// expandReplies fills thread.Snippet.Replies.Comments with every reply when the
// inlined slice (capped at 5 by the API) is shorter than TotalReplyCount. When
// no fetch is needed it leaves the thread untouched.
func expandReplies(ctx context.Context, svc *youtube.Service, thread *youtube.CommentThread) error {
	if thread == nil || thread.Snippet == nil {
		return nil
	}
	total := thread.Snippet.TotalReplyCount
	have := int64(0)
	if thread.Replies != nil {
		have = int64(len(thread.Replies.Comments))
	}
	if total <= have {
		return nil
	}

	var all []*youtube.Comment
	pageToken := ""
	for {
		call := svc.Comments.List([]string{"id", "snippet"}).
			ParentId(thread.Id).
			TextFormat("plainText").
			MaxResults(100)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("comments.list: %w", err)
		}
		all = append(all, resp.Items...)
		if resp.NextPageToken == "" || len(resp.Items) == 0 {
			break
		}
		pageToken = resp.NextPageToken
	}

	if thread.Replies == nil {
		thread.Replies = &youtube.CommentThreadReplies{}
	}
	thread.Replies.Comments = all
	return nil
}

func buildThreadRows(thread *youtube.CommentThread) [][]string {
	rows := make([][]string, 0, 1+replyCount(thread))

	top := topLevelComment(thread)
	if top != nil {
		rows = append(rows, []string{
			top.Id,
			commentAuthor(top),
			commentPublished(top),
			commentLikes(top),
			commentText(top),
		})
	}

	for _, reply := range replies(thread) {
		rows = append(rows, []string{
			reply.Id,
			"  ↳ " + commentAuthor(reply),
			commentPublished(reply),
			commentLikes(reply),
			commentText(reply),
		})
	}

	return rows
}

func topLevelComment(t *youtube.CommentThread) *youtube.Comment {
	if t.Snippet == nil {
		return nil
	}
	return t.Snippet.TopLevelComment
}

func replies(t *youtube.CommentThread) []*youtube.Comment {
	if t == nil || t.Replies == nil {
		return nil
	}
	return t.Replies.Comments
}

func replyCount(t *youtube.CommentThread) int {
	return len(replies(t))
}

func commentAuthor(c *youtube.Comment) string {
	if c == nil || c.Snippet == nil {
		return ""
	}
	return c.Snippet.AuthorDisplayName
}

func commentPublished(c *youtube.Comment) string {
	if c == nil || c.Snippet == nil {
		return ""
	}
	if ts, err := time.Parse(time.RFC3339, c.Snippet.PublishedAt); err == nil {
		return ts.UTC().Format("2006-01-02")
	}
	return c.Snippet.PublishedAt
}

func commentLikes(c *youtube.Comment) string {
	if c == nil || c.Snippet == nil {
		return "0"
	}
	return fmt.Sprintf("%d", c.Snippet.LikeCount)
}

func commentText(c *youtube.Comment) string {
	if c == nil || c.Snippet == nil {
		return ""
	}
	return collapseWhitespace(c.Snippet.TextDisplay)
}
