package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newPlaylistsDeleteCmd() *cobra.Command {
	var yes bool

	totalCost := ytapi.CostList + ytapi.CostDelete

	c := &cobra.Command{
		Use:   "delete <id>",
		Short: fmt.Sprintf("Delete a playlist (cost: %d units)", ytapi.CostDelete),
		Long: fmt.Sprintf("Delete a playlist owned by the authenticated user.\n\n"+
			"Quota cost: %d unit (playlists.list, for the confirmation lookup) + %d units (playlists.delete) = %d units.\n"+
			"Prompts for confirmation unless --yes is provided.\n"+
			"Use --dry-run to print the planned mutation without calling the API.",
			ytapi.CostList, ytapi.CostDelete, totalCost),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		id := args[0]

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		resp, err := svc.Playlists.
			List([]string{"id", "snippet"}).
			Id(id).
			MaxResults(1).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("playlists.list: %w", err)
		}
		if len(resp.Items) == 0 {
			return fmt.Errorf("playlist %q not found", id)
		}
		title := ""
		if resp.Items[0].Snippet != nil {
			title = resp.Items[0].Snippet.Title
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostDelete,
				"would delete playlist id=%s title=%q",
				id, title,
			)
			return nil
		}

		if !yes {
			fmt.Fprintf(cmd.OutOrStderr(),
				"About to delete playlist id=%s title=%q. This cannot be undone.\nType 'yes' to confirm: ",
				id, title,
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

		if err := svc.Playlists.Delete(id).Context(ctx).Do(); err != nil {
			return fmt.Errorf("playlists.delete: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Deleted playlist id=%s title=%q\n", id, title)
		return nil
	}

	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}
