package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newVideosDeleteCmd() *cobra.Command {
	var yes bool

	totalCost := ytapi.CostList + ytapi.CostDelete

	c := &cobra.Command{
		Use:   "delete <id-or-url>",
		Short: fmt.Sprintf("Delete a video (cost: %d units)", ytapi.CostDelete),
		Long: fmt.Sprintf("Delete a video owned by the authenticated user.\n\n"+
			"The argument may be a raw video id (e.g. dQw4w9WgXcQ) or a YouTube URL\n"+
			"(watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Quota cost: %d unit (videos.list, for the confirmation lookup) + %d units (videos.delete) = %d units.\n"+
			"Prompts for confirmation unless --yes is provided.\n"+
			"Use --dry-run to print the planned mutation without calling the API.\n\n"+
			"Note: only videos owned by the authenticated account can be deleted; the\n"+
			"YouTube API returns 403/forbidden for someone else's video.",
			ytapi.CostList, ytapi.CostDelete, totalCost),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		videoID, err := parseVideoID(args[0])
		if err != nil {
			return fmt.Errorf("invalid video reference %q: %w", args[0], err)
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		resp, err := svc.Videos.
			List([]string{"id", "snippet"}).
			Id(videoID).
			MaxResults(1).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("videos.list: %w", err)
		}
		if len(resp.Items) == 0 {
			return fmt.Errorf("video %q not found", videoID)
		}
		title := ""
		if resp.Items[0].Snippet != nil {
			title = resp.Items[0].Snippet.Title
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostDelete,
				"would delete video id=%s title=%q",
				videoID, title,
			)
			return nil
		}

		if !yes {
			fmt.Fprintf(cmd.OutOrStderr(),
				"About to delete video id=%s title=%q. This cannot be undone.\nType 'yes' to confirm: ",
				videoID, title,
			)
			reader := bufio.NewReader(cmd.InOrStdin())
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read confirmation: %w", err)
			}
			if strings.TrimSpace(line) != "yes" {
				return fmt.Errorf("aborted")
			}
		}

		if err := svc.Videos.Delete(videoID).Context(ctx).Do(); err != nil {
			return fmt.Errorf("videos.delete: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted video id=%s title=%q\n", videoID, title)
		return nil
	}

	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}
