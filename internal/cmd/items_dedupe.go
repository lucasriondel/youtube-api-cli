package cmd

import (
	"bufio"
	"fmt"
	"sort"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newItemsDedupeCmd() *cobra.Command {
	var (
		keep string
		yes  bool
	)

	c := &cobra.Command{
		Use:   "dedupe <playlist-id>",
		Short: "Remove duplicate videoIds from a playlist",
		Long: fmt.Sprintf("Detect items that point to the same videoId and remove duplicates.\n\n"+
			"By default the earliest occurrence (lowest position) is kept and the rest are\n"+
			"deleted. Pass --keep=last to keep the latest occurrence instead.\n\n"+
			"Quota cost: %d unit per page of items.list (etag-cached) + %d units per\n"+
			"duplicate item removed.\n\n"+
			"Use --dry-run to print the plan without calling the delete API.\n"+
			"Prompts for confirmation if the estimated cost exceeds 1000 units, unless --yes.",
			ytapi.CostList, ytapi.CostDelete),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		playlistID := strings.TrimSpace(args[0])
		if playlistID == "" {
			return fmt.Errorf("empty playlist id")
		}
		keepMode, err := parseKeepMode(keep)
		if err != nil {
			return err
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		items, err := fetchAllPlaylistItems(ctx, svc, playlistID)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Fprintf(cmd.OutOrStderr(), "playlist %s is empty — nothing to dedupe\n", playlistID)
			return nil
		}

		dupes := planDedupe(items, keepMode)
		writeCost := len(dupes) * ytapi.CostDelete

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), writeCost,
				"would remove %d duplicate item(s) from playlist %s (keep=%s)",
				len(dupes), playlistID, keepMode,
			)
			return renderDupes(cmd, dupes)
		}

		if len(dupes) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "no duplicates found in playlist %s\n", playlistID)
			return nil
		}

		if writeCost > 1000 && !yes {
			fmt.Fprintf(cmd.OutOrStderr(),
				"Estimated cost is %d units (%d deletes × %d). Type 'yes' to proceed: ",
				writeCost, len(dupes), ytapi.CostDelete,
			)
			reader := bufio.NewReader(cmd.InOrStdin())
			line, rerr := reader.ReadString('\n')
			if rerr != nil {
				return fmt.Errorf("read confirmation: %w", rerr)
			}
			if strings.TrimSpace(line) != "yes" {
				return fmt.Errorf("aborted")
			}
		}

		for _, d := range dupes {
			if err := svc.PlaylistItems.Delete(d.ItemID).Context(ctx).Do(); err != nil {
				return fmt.Errorf("playlistItems.delete (itemId=%s): %w", d.ItemID, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed duplicate item id=%s videoId=%s\n", d.ItemID, d.VideoID)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Deduped: %d item(s) removed, %d units spent.\n", len(dupes), writeCost)
		return nil
	}

	c.Flags().StringVar(&keep, "keep", "first", "which occurrence to keep when duplicates are found: first|last")
	c.Flags().BoolVar(&yes, "yes", false, "skip the >1000-unit confirmation prompt")
	return c
}

type keepMode string

const (
	keepFirst keepMode = "first"
	keepLast  keepMode = "last"
)

func parseKeepMode(s string) (keepMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "first":
		return keepFirst, nil
	case "last":
		return keepLast, nil
	default:
		return "", fmt.Errorf("invalid --keep %q (want one of: first, last)", s)
	}
}

type duplicateItem struct {
	ItemID   string
	VideoID  string
	Title    string
	Position int64
	KeptID   string
}

// planDedupe groups items by videoId, picks one to keep per group based on
// `keep`, and returns the rest as deletion candidates. Items missing a videoId
// are left alone (we can't tell whether they're duplicates of anything).
// The returned slice is sorted by Position ascending so the dry-run table
// reads top-to-bottom in playlist order.
func planDedupe(items []*youtube.PlaylistItem, keep keepMode) []duplicateItem {
	type indexed struct {
		item *youtube.PlaylistItem
		pos  int64
	}
	groups := make(map[string][]indexed)
	for _, it := range items {
		vid := videoIDOf(it)
		if vid == "" {
			continue
		}
		var pos int64
		if it.Snippet != nil {
			pos = it.Snippet.Position
		}
		groups[vid] = append(groups[vid], indexed{item: it, pos: pos})
	}

	dupes := make([]duplicateItem, 0)
	for vid, group := range groups {
		if len(group) < 2 {
			continue
		}
		sort.SliceStable(group, func(i, j int) bool { return group[i].pos < group[j].pos })

		var keptIdx int
		if keep == keepLast {
			keptIdx = len(group) - 1
		}
		keptID := group[keptIdx].item.Id

		for i, g := range group {
			if i == keptIdx {
				continue
			}
			dupes = append(dupes, duplicateItem{
				ItemID:   g.item.Id,
				VideoID:  vid,
				Title:    snippetTitle(g.item),
				Position: g.pos,
				KeptID:   keptID,
			})
		}
	}

	sort.SliceStable(dupes, func(i, j int) bool { return dupes[i].Position < dupes[j].Position })
	return dupes
}

func renderDupes(cmd *cobra.Command, dupes []duplicateItem) error {
	rows := make([][]string, 0, len(dupes))
	for _, d := range dupes {
		rows = append(rows, []string{
			fmt.Sprintf("%d", d.Position),
			d.ItemID,
			d.VideoID,
			d.KeptID,
			d.Title,
		})
	}
	format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
	return output.Render(
		cmd.OutOrStdout(),
		format,
		[]string{"POS", "ITEM_ID", "VIDEO_ID", "KEPT_ID", "TITLE"},
		rows,
		dupes,
	)
}
