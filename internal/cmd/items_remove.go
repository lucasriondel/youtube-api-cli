package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newItemsRemoveCmd() *cobra.Command {
	var yes bool

	c := &cobra.Command{
		Use:   "remove <item-id>...",
		Short: fmt.Sprintf("Remove one or more items from a playlist (cost: %d units per item)", ytapi.CostDelete),
		Long: fmt.Sprintf("Remove items from a playlist by playlistItemId.\n\n"+
			"Note: this takes the playlistItemId (the ITEM_ID column from `yt items list`),\n"+
			"NOT the videoId. The same video can appear multiple times in a playlist with\n"+
			"distinct playlistItemIds, so the API requires the item id to disambiguate.\n\n"+
			"Quota cost: %d units per item removed.\n"+
			"Prompts for confirmation unless --yes is provided.\n"+
			"Use --dry-run to print the planned mutations without calling the API.", ytapi.CostDelete),
		Args: cobra.MinimumNArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		itemIDs := make([]string, 0, len(args))
		for _, raw := range args {
			id := strings.TrimSpace(raw)
			if id == "" {
				return fmt.Errorf("empty item id")
			}
			itemIDs = append(itemIDs, id)
		}

		totalCost := ytapi.CostDelete * len(itemIDs)

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), totalCost,
				"would remove %d item(s)",
				len(itemIDs),
			)
			for _, id := range itemIDs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", id)
			}
			return nil
		}

		if !yes {
			fmt.Fprintf(cmd.OutOrStderr(),
				"About to remove %d item(s) from their playlist(s). This cannot be undone.\nType 'yes' to confirm: ",
				len(itemIDs),
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

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		for _, id := range itemIDs {
			if err := svc.PlaylistItems.Delete(id).Context(ctx).Do(); err != nil {
				return fmt.Errorf("playlistItems.delete (itemId=%s): %w", id, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed item id=%s\n", id)
		}

		return nil
	}

	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}
