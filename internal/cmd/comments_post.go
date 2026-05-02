package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newCommentsPostCmd() *cobra.Command {
	var (
		videoID string
		text    string
	)

	c := &cobra.Command{
		Use:   "post --video <id-or-url> --text <text>",
		Short: fmt.Sprintf("Post a new top-level comment thread on a video (cost: %d units)", ytapi.CostInsert),
		Long: fmt.Sprintf("Create a new top-level comment thread on a video via commentThreads.insert.\n\n"+
			"Quota cost: %d units per call.\n\n"+
			"--video accepts a raw video id (e.g. dQw4w9WgXcQ) or a YouTube URL\n"+
			"(watch?v=..., youtu.be/..., shorts/...).\n"+
			"--text is the comment body. Pass plain text; YouTube applies its own\n"+
			"formatting on render.\n\n"+
			"Use --dry-run to print the planned mutation without calling the API.\n\n"+
			"On success the new thread id is printed on stdout (or the raw\n"+
			"youtube#commentThread object with --json).",
			ytapi.CostInsert),
		Args: cobra.NoArgs,
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(videoID) == "" {
			return fmt.Errorf("--video is required")
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("--text is required and cannot be empty")
		}

		id, err := parseVideoID(videoID)
		if err != nil {
			return fmt.Errorf("invalid --video %q: %w", videoID, err)
		}

		thread := &youtube.CommentThread{
			Snippet: &youtube.CommentThreadSnippet{
				VideoId: id,
				TopLevelComment: &youtube.Comment{
					Snippet: &youtube.CommentSnippet{
						TextOriginal: text,
					},
				},
			},
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostInsert,
				"would post comment on video %s (text=%q)", id, truncateForLog(text, 80))
			return nil
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		created, err := svc.CommentThreads.
			Insert([]string{"snippet"}, thread).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("commentThreads.insert: %w", err)
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"THREAD_ID", created.Id},
				{"VIDEO_ID", id},
				{"AUTHOR", commentThreadAuthor(created)},
				{"TEXT", commentThreadText(created)},
			},
			created,
		)
	}

	c.Flags().StringVar(&videoID, "video", "", "video to comment on (raw id or YouTube URL) (required)")
	c.Flags().StringVar(&text, "text", "", "comment body (required)")
	return c
}

// truncateForLog shortens s to maxLen runes for diagnostic output, appending an
// ellipsis if it was clipped. Used only in --dry-run / error messages — the
// real API payload is never truncated.
func truncateForLog(s string, maxLen int) string {
	r := []rune(s)
	if len(r) <= maxLen {
		return s
	}
	return string(r[:maxLen]) + "..."
}
