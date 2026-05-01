package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newSubsRemoveCmd() *cobra.Command {
	var yes bool

	c := &cobra.Command{
		Use:   "remove <subscription-id>...",
		Short: fmt.Sprintf("Unsubscribe by subscription resource id (cost: %d units per id)", ytapi.CostDelete),
		Long: fmt.Sprintf("Unsubscribe via subscriptions.delete.\n\n"+
			"Note: this takes the subscription resource id (the SUB_ID column from\n"+
			"`yt subs list`), NOT a channel id. The subscription id is the per-account\n"+
			"identifier the API issues when you subscribe; channel ids (UC...) are not\n"+
			"accepted.\n\n"+
			"Quota cost: %d units per subscription removed.\n"+
			"Prompts for confirmation unless --yes is provided.\n"+
			"Use --dry-run to print the planned mutations without calling the API.",
			ytapi.CostDelete),
		Args: cobra.MinimumNArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		subIDs := make([]string, 0, len(args))
		for _, raw := range args {
			id := strings.TrimSpace(raw)
			if id == "" {
				return fmt.Errorf("invalid subscription id %q: empty value", raw)
			}
			subIDs = append(subIDs, id)
		}

		totalCost := ytapi.CostDelete * len(subIDs)

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), totalCost,
				"would unsubscribe from %d subscription(s)",
				len(subIDs),
			)
			for _, id := range subIDs {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", id)
			}
			return nil
		}

		if !yes {
			fmt.Fprintf(cmd.OutOrStderr(),
				"About to remove %d subscription(s). This cannot be undone.\nType 'yes' to confirm: ",
				len(subIDs),
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

		for _, id := range subIDs {
			if err := svc.Subscriptions.Delete(id).Context(ctx).Do(); err != nil {
				return fmt.Errorf("subscriptions.delete (subId=%s): %w", id, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed subscription id=%s\n", id)
		}

		return nil
	}

	c.Flags().BoolVar(&yes, "yes", false, "skip the confirmation prompt")
	return c
}
