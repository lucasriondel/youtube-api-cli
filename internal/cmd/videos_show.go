package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newVideosShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id-or-url>...",
		Short: "Show metadata for one or more videos (cost: 1 unit per batch of 50)",
		Long: "Show metadata for one or more videos. Each argument may be a raw video id\n" +
			"(e.g. dQw4w9WgXcQ) or a YouTube URL (watch?v=..., youtu.be/..., shorts/...).\n\n" +
			"Fetches snippet,contentDetails,statistics,status in a single batched videos.list\n" +
			"call (up to 50 ids per batch).\n\n" +
			"Quota cost: 1 unit per batch of 50 ids.",
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

			var all []*youtube.Video
			for i := 0; i < len(ids); i += 50 {
				end := i + 50
				if end > len(ids) {
					end = len(ids)
				}
				batch := ids[i:end]
				resp, err := svc.Videos.
					List([]string{"id", "snippet", "contentDetails", "statistics", "status"}).
					Id(strings.Join(batch, ",")).
					MaxResults(50).
					Context(ctx).
					Do()
				if err != nil {
					return fmt.Errorf("videos.list: %w", err)
				}
				all = append(all, resp.Items...)
			}

			returned := make(map[string]bool, len(all))
			for _, v := range all {
				returned[v.Id] = true
			}
			missing := make([]string, 0)
			for _, id := range ids {
				if !returned[id] {
					missing = append(missing, id)
				}
			}
			if len(missing) > 0 {
				fmt.Fprintf(cmd.OutOrStderr(),
					"warning: %d video(s) not found or unavailable: %s\n",
					len(missing), strings.Join(missing, ", "),
				)
			}

			rows := make([][]string, 0, len(all))
			for _, v := range all {
				rows = append(rows, []string{
					v.Id,
					videoTitle(v),
					videoChannel(v),
					videoDuration(v),
					videoViews(v),
					videoLikes(v),
					videoPublished(v),
					videoPrivacy(v),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"ID", "TITLE", "CHANNEL", "DURATION", "VIEWS", "LIKES", "PUBLISHED", "PRIVACY"},
				rows,
				all,
			)
		},
	}
}

func videoTitle(v *youtube.Video) string {
	if v.Snippet == nil {
		return ""
	}
	return v.Snippet.Title
}

func videoChannel(v *youtube.Video) string {
	if v.Snippet == nil {
		return ""
	}
	return v.Snippet.ChannelTitle
}

func videoDuration(v *youtube.Video) string {
	if v.ContentDetails == nil {
		return ""
	}
	return formatISODuration(v.ContentDetails.Duration)
}

func videoViews(v *youtube.Video) string {
	if v.Statistics == nil {
		return ""
	}
	return fmt.Sprintf("%d", v.Statistics.ViewCount)
}

func videoLikes(v *youtube.Video) string {
	if v.Statistics == nil {
		return ""
	}
	return fmt.Sprintf("%d", v.Statistics.LikeCount)
}

func videoPublished(v *youtube.Video) string {
	if v.Snippet == nil {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, v.Snippet.PublishedAt); err == nil {
		return t.UTC().Format("2006-01-02")
	}
	return v.Snippet.PublishedAt
}

func videoPrivacy(v *youtube.Video) string {
	if v.Status == nil {
		return ""
	}
	return v.Status.PrivacyStatus
}

// formatISODuration converts an ISO-8601 PT#H#M#S duration into H:MM:SS / M:SS.
// Falls back to the raw input on parse failure.
func formatISODuration(s string) string {
	if !strings.HasPrefix(s, "PT") {
		return s
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
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, sec)
	}
	return fmt.Sprintf("%d:%02d", m, sec)
}
