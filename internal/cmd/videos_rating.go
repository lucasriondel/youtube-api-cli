package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newVideosRatingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rating <id-or-url>...",
		Short: fmt.Sprintf("Show the authorized user's rating for one or more videos (cost: %d unit per batch of 50)", ytapi.CostGetRating),
		Long: fmt.Sprintf("Show the rating ('like', 'dislike', or 'none') that the authorized user has\n"+
			"assigned to one or more videos. Each argument may be a raw video id\n"+
			"(e.g. dQw4w9WgXcQ) or a YouTube URL (watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Issues a single batched videos.getRating call per 50 ids.\n\n"+
			"Quota cost: %d unit per batch of 50 ids.", ytapi.CostGetRating),
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ids := make([]string, 0, len(args))
			for _, raw := range args {
				id, err := parseVideoID(raw)
				if err != nil {
					return fmt.Errorf("invalid video reference %q: %w", raw, err)
				}
				ids = append(ids, id)
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			var all []*youtube.VideoRating
			for i := 0; i < len(ids); i += 50 {
				end := i + 50
				if end > len(ids) {
					end = len(ids)
				}
				batch := ids[i:end]
				resp, err := svc.Videos.GetRating(batch).Context(ctx).Do()
				if err != nil {
					return fmt.Errorf("videos.getRating: %w", err)
				}
				all = append(all, resp.Items...)
			}

			returned := make(map[string]bool, len(all))
			for _, r := range all {
				returned[r.VideoId] = true
			}
			missing := make([]string, 0)
			for _, id := range ids {
				if !returned[id] {
					missing = append(missing, id)
				}
			}
			if len(missing) > 0 {
				fmt.Fprintf(cmd.OutOrStderr(),
					"warning: %d video(s) not returned by API: %s\n",
					len(missing), strings.Join(missing, ", "),
				)
			}

			rows := make([][]string, 0, len(all))
			for _, r := range all {
				rows = append(rows, []string{r.VideoId, r.Rating})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"VIDEO_ID", "RATING"},
				rows,
				all,
			)
		},
	}
}
