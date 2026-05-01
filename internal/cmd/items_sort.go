package cmd

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newItemsSortCmd() *cobra.Command {
	var (
		by      string
		reverse bool
		yes     bool
	)

	c := &cobra.Command{
		Use:   "sort <playlist-id>",
		Short: "Sort items in a playlist by title|date|duration|channel",
		Long: fmt.Sprintf("Sort the items in a playlist locally and apply the new order via "+
			"playlistItems.update. Only items whose position actually changes incur a write.\n\n"+
			"Sort keys:\n"+
			"  title    — case-insensitive snippet.title\n"+
			"  date     — snippet.publishedAt (ascending = oldest first)\n"+
			"  duration — videos.list contentDetails.duration (requires extra reads)\n"+
			"  channel  — case-insensitive snippet.videoOwnerChannelTitle\n\n"+
			"Quota cost: %d unit per page of items.list (etag-cached) + %d unit per batch of\n"+
			"50 ids for videos.list (only when --by=duration) + %d units per moved item.\n"+
			"A 50-item playlist sorted with every item moving costs 1 + 50×%d = %d units.\n\n"+
			"--dry-run is mandatory the first time: re-run without --dry-run to apply.\n"+
			"Prompts for confirmation if the estimated cost exceeds 1000 units, unless --yes.",
			ytapi.CostList, ytapi.CostList, ytapi.CostUpdate,
			ytapi.CostUpdate, 1+50*ytapi.CostUpdate),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		playlistID := strings.TrimSpace(args[0])
		if playlistID == "" {
			return fmt.Errorf("empty playlist id")
		}
		key, err := parseSortKey(by)
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
			fmt.Fprintf(cmd.OutOrStderr(), "playlist %s is empty — nothing to sort\n", playlistID)
			return nil
		}

		var durations map[string]time.Duration
		if key == sortByDuration {
			durations, err = fetchVideoDurations(ctx, svc, items)
			if err != nil {
				return err
			}
		}

		sorted := make([]*youtube.PlaylistItem, len(items))
		copy(sorted, items)
		sort.SliceStable(sorted, func(i, j int) bool {
			return lessByKey(sorted[i], sorted[j], key, durations, reverse)
		})

		moves := planMoves(items, sorted)
		// Apply low-to-high target position so each insert doesn't shift
		// already-placed items. PRD doesn't specify order but this minimizes
		// surprises on partial failures.
		sort.SliceStable(moves, func(i, j int) bool { return moves[i].To < moves[j].To })
		writeCost := len(moves) * ytapi.CostUpdate

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), writeCost,
				"would move %d of %d item(s) to sort playlist %s by %s%s",
				len(moves), len(items), playlistID, by, reverseSuffix(reverse),
			)
			return renderMoves(cmd, moves)
		}

		if len(moves) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "playlist already sorted by %s%s — no changes\n", by, reverseSuffix(reverse))
			return nil
		}

		if writeCost > 1000 && !yes {
			fmt.Fprintf(cmd.OutOrStderr(),
				"Estimated cost is %d units (%d moves × %d). Type 'yes' to proceed: ",
				writeCost, len(moves), ytapi.CostUpdate,
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

		for _, m := range moves {
			if err := applyMove(ctx, svc, playlistID, m); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Moved item id=%s from %d to %d\n", m.ItemID, m.From, m.To)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Sorted: %d move(s), %d units spent.\n", len(moves), writeCost)
		return nil
	}

	c.Flags().StringVar(&by, "by", "", "sort key: title|date|duration|channel (required)")
	c.Flags().BoolVar(&reverse, "reverse", false, "reverse sort order")
	c.Flags().BoolVar(&yes, "yes", false, "skip the >1000-unit confirmation prompt")
	_ = c.MarkFlagRequired("by")
	return c
}

type sortKey int

const (
	sortByTitle sortKey = iota
	sortByDate
	sortByDuration
	sortByChannel
)

func parseSortKey(s string) (sortKey, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "title":
		return sortByTitle, nil
	case "date":
		return sortByDate, nil
	case "duration":
		return sortByDuration, nil
	case "channel":
		return sortByChannel, nil
	default:
		return 0, fmt.Errorf("invalid --by %q (want one of: title, date, duration, channel)", s)
	}
}

func reverseSuffix(rev bool) string {
	if rev {
		return " (reversed)"
	}
	return ""
}

// fetchAllPlaylistItems pulls every page of a playlist via the cached helper.
func fetchAllPlaylistItems(ctx context.Context, svc *youtube.Service, playlistID string) ([]*youtube.PlaylistItem, error) {
	parts := []string{"id", "snippet", "contentDetails"}
	var all []*youtube.PlaylistItem
	pageToken := ""
	for {
		resp, err := fetchPlaylistItemsPage(ctx, svc, playlistID, parts, pageToken, false)
		if err != nil {
			return nil, err
		}
		all = append(all, resp.Items...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return all, nil
}

// fetchVideoDurations pulls contentDetails.duration for every videoId in items
// via batched videos.list calls (1 unit per 50 ids).
func fetchVideoDurations(ctx context.Context, svc *youtube.Service, items []*youtube.PlaylistItem) (map[string]time.Duration, error) {
	ids := make([]string, 0, len(items))
	seen := make(map[string]bool)
	for _, it := range items {
		vid := videoIDOf(it)
		if vid == "" || seen[vid] {
			continue
		}
		seen[vid] = true
		ids = append(ids, vid)
	}

	out := make(map[string]time.Duration, len(ids))
	for i := 0; i < len(ids); i += 50 {
		end := i + 50
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]
		resp, err := svc.Videos.
			List([]string{"id", "contentDetails"}).
			Id(strings.Join(batch, ",")).
			MaxResults(50).
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("videos.list (durations): %w", err)
		}
		for _, v := range resp.Items {
			if v.ContentDetails != nil {
				out[v.Id] = parseISODurationSeconds(v.ContentDetails.Duration)
			}
		}
	}
	return out, nil
}

func videoIDOf(it *youtube.PlaylistItem) string {
	if it.Snippet != nil && it.Snippet.ResourceId != nil && it.Snippet.ResourceId.VideoId != "" {
		return it.Snippet.ResourceId.VideoId
	}
	if it.ContentDetails != nil {
		return it.ContentDetails.VideoId
	}
	return ""
}

// parseISODurationSeconds turns PT#H#M#S into a time.Duration. Unknown / empty
// durations (e.g. live streams without contentDetails.duration) become 0.
func parseISODurationSeconds(s string) time.Duration {
	if !strings.HasPrefix(s, "PT") {
		return 0
	}
	rest := s[2:]
	var h, m, sec int
	cur := ""
	for _, r := range rest {
		switch r {
		case 'H':
			fmt.Sscanf(cur, "%d", &h)
			cur = ""
		case 'M':
			fmt.Sscanf(cur, "%d", &m)
			cur = ""
		case 'S':
			fmt.Sscanf(cur, "%d", &sec)
			cur = ""
		default:
			cur += string(r)
		}
	}
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second
}

func lessByKey(a, b *youtube.PlaylistItem, key sortKey, durations map[string]time.Duration, reverse bool) bool {
	res := compareKey(a, b, key, durations)
	if reverse {
		return res > 0
	}
	return res < 0
}

// compareKey returns -1/0/+1. Falls back to ITEM_ID for stable ordering when
// the primary key ties, so sort.SliceStable yields a deterministic result.
func compareKey(a, b *youtube.PlaylistItem, key sortKey, durations map[string]time.Duration) int {
	switch key {
	case sortByTitle:
		return cmpStr(snippetTitle(a), snippetTitle(b), true, a.Id, b.Id)
	case sortByChannel:
		return cmpStr(snippetChannel(a), snippetChannel(b), true, a.Id, b.Id)
	case sortByDate:
		return cmpStr(snippetPublished(a), snippetPublished(b), false, a.Id, b.Id)
	case sortByDuration:
		da := durations[videoIDOf(a)]
		db := durations[videoIDOf(b)]
		if da < db {
			return -1
		}
		if da > db {
			return 1
		}
		return cmpStr(a.Id, b.Id, false, "", "")
	}
	return 0
}

func cmpStr(a, b string, fold bool, tieA, tieB string) int {
	la, lb := a, b
	if fold {
		la, lb = strings.ToLower(a), strings.ToLower(b)
	}
	if la < lb {
		return -1
	}
	if la > lb {
		return 1
	}
	if tieA < tieB {
		return -1
	}
	if tieA > tieB {
		return 1
	}
	return 0
}

func snippetTitle(it *youtube.PlaylistItem) string {
	if it.Snippet == nil {
		return ""
	}
	return it.Snippet.Title
}

func snippetChannel(it *youtube.PlaylistItem) string {
	if it.Snippet == nil {
		return ""
	}
	return it.Snippet.VideoOwnerChannelTitle
}

func snippetPublished(it *youtube.PlaylistItem) string {
	if it.Snippet == nil {
		return ""
	}
	return it.Snippet.PublishedAt
}

type plannedMove struct {
	ItemID     string
	VideoID    string
	Title      string
	From       int64
	To         int64
	ResourceID *youtube.ResourceId
}

// planMoves returns the moves required to turn `original` into `sorted`. We
// only emit a move for items whose position actually changes — keeping write
// cost proportional to the number of out-of-place items, not the playlist size.
func planMoves(original, sorted []*youtube.PlaylistItem) []plannedMove {
	moves := make([]plannedMove, 0)
	for newPos, it := range sorted {
		var from int64
		var resID *youtube.ResourceId
		if it.Snippet != nil {
			from = it.Snippet.Position
			resID = it.Snippet.ResourceId
		}
		to := int64(newPos)
		if from == to {
			continue
		}
		moves = append(moves, plannedMove{
			ItemID:     it.Id,
			VideoID:    videoIDOf(it),
			Title:      snippetTitle(it),
			From:       from,
			To:         to,
			ResourceID: resID,
		})
	}
	return moves
}

func renderMoves(cmd *cobra.Command, moves []plannedMove) error {
	rows := make([][]string, 0, len(moves))
	for _, m := range moves {
		rows = append(rows, []string{
			fmt.Sprintf("%d", m.From),
			fmt.Sprintf("%d", m.To),
			m.ItemID,
			m.VideoID,
			m.Title,
		})
	}
	format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
	return output.Render(
		cmd.OutOrStdout(),
		format,
		[]string{"FROM", "TO", "ITEM_ID", "VIDEO_ID", "TITLE"},
		rows,
		moves,
	)
}

func applyMove(ctx context.Context, svc *youtube.Service, playlistID string, m plannedMove) error {
	if m.ResourceID == nil {
		return fmt.Errorf("item %s missing resourceId — cannot update", m.ItemID)
	}
	updated := &youtube.PlaylistItem{
		Id: m.ItemID,
		Snippet: &youtube.PlaylistItemSnippet{
			PlaylistId: playlistID,
			ResourceId: m.ResourceID,
			Position:   m.To,
		},
	}
	if m.To == 0 {
		updated.Snippet.ForceSendFields = []string{"Position"}
	}
	if _, err := svc.PlaylistItems.Update([]string{"snippet"}, updated).Context(ctx).Do(); err != nil {
		return fmt.Errorf("playlistItems.update (itemId=%s): %w", m.ItemID, err)
	}
	return nil
}
