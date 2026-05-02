package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newCommentsReplyCmd() *cobra.Command {
	var text string

	c := &cobra.Command{
		Use:   "reply <parent-id> --text <text>",
		Short: fmt.Sprintf("Reply to an existing comment (cost: %d units)", ytapi.CostInsert),
		Long: fmt.Sprintf("Post a reply to an existing comment via comments.insert.\n\n"+
			"Quota cost: %d units per call.\n\n"+
			"<parent-id> is the id of the comment being replied to. For a reply on\n"+
			"a top-level thread this is the THREAD_ID column from `comments list`\n"+
			"(also returned by `comments post`). To reply to another reply, pass\n"+
			"its COMMENT_ID — YouTube flattens reply chains, so the new comment\n"+
			"will be attached to the same thread regardless of which reply id you\n"+
			"target.\n\n"+
			"--text is the reply body. Pass plain text; YouTube applies its own\n"+
			"formatting on render.\n\n"+
			"Use --dry-run to print the planned mutation without calling the API.\n\n"+
			"On success the new comment id is printed on stdout (or the raw\n"+
			"youtube#comment object with --json).",
			ytapi.CostInsert),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		parentID := strings.TrimSpace(args[0])
		if parentID == "" {
			return fmt.Errorf("invalid parent id: empty value")
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("--text is required and cannot be empty")
		}

		comment := &youtube.Comment{
			Snippet: &youtube.CommentSnippet{
				ParentId:     parentID,
				TextOriginal: text,
			},
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostInsert,
				"would reply to comment %s (text=%q)", parentID, truncateForLog(text, 80))
			return nil
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		created, err := svc.Comments.
			Insert([]string{"snippet"}, comment).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("comments.insert: %w", err)
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"COMMENT_ID", created.Id},
				{"PARENT_ID", parentID},
				{"AUTHOR", commentAuthor(created)},
				{"TEXT", commentText(created)},
			},
			created,
		)
	}

	c.Flags().StringVar(&text, "text", "", "reply body (required)")
	return c
}
