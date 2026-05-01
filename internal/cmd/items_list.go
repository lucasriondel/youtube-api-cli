package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/lucasrndl/yt/internal/cache"
	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/youtube/v3"
)

func newItemsListCmd() *cobra.Command {
	var noCache bool
	c := &cobra.Command{
		Use:   "list <playlist-id>",
		Short: "List the items in a playlist",
		Long: "List the items in a playlist.\n\n" +
			"Cost: 1 unit per page. Responses are cached on disk by etag, so an unchanged " +
			"playlist costs 0 quota units to refetch (the API returns 304 Not Modified). " +
			"Pass --no-cache to bypass the cache entirely.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			playlistID := args[0]
			parts := []string{"id", "snippet", "contentDetails", "status"}

			var all []*youtube.PlaylistItem
			pageToken := ""
			for {
				resp, err := fetchPlaylistItemsPage(ctx, svc, playlistID, parts, pageToken, noCache)
				if err != nil {
					return err
				}
				all = append(all, resp.Items...)
				if resp.NextPageToken == "" {
					break
				}
				pageToken = resp.NextPageToken
			}

			rows := make([][]string, 0, len(all))
			for _, it := range all {
				position, videoID, title, channel := "", "", "", ""
				if it.Snippet != nil {
					position = fmt.Sprintf("%d", it.Snippet.Position)
					title = it.Snippet.Title
					channel = it.Snippet.VideoOwnerChannelTitle
					if it.Snippet.ResourceId != nil {
						videoID = it.Snippet.ResourceId.VideoId
					}
				}
				if videoID == "" && it.ContentDetails != nil {
					videoID = it.ContentDetails.VideoId
				}
				rows = append(rows, []string{position, it.Id, videoID, title, channel})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"POS", "ITEM_ID", "VIDEO_ID", "TITLE", "CHANNEL"},
				rows,
				all,
			)
		},
	}
	c.Flags().BoolVar(&noCache, "no-cache", false, "skip the on-disk etag cache (always re-fetch)")
	return c
}

// fetchPlaylistItemsPage fetches one page of playlistItems.list, using the
// etag cache to avoid re-spending quota when the server returns 304.
func fetchPlaylistItemsPage(ctx context.Context, svc *youtube.Service, playlistID string, parts []string, pageToken string, noCache bool) (*youtube.PlaylistItemListResponse, error) {
	key := cache.Key("playlistItems.list", map[string]string{
		"playlistId": playlistID,
		"parts":      strings.Join(parts, ","),
		"pageToken":  pageToken,
		"maxResults": "50",
	})

	var cached *cache.Entry
	if !noCache {
		entry, err := cache.Lookup(key)
		if err != nil && !errors.Is(err, cache.ErrMiss) {
			fmt.Fprintf(os.Stderr, "warning: cache lookup failed: %v\n", err)
		} else if err == nil {
			cached = entry
		}
	}

	call := svc.PlaylistItems.List(parts).
		PlaylistId(playlistID).
		MaxResults(50)
	if pageToken != "" {
		call = call.PageToken(pageToken)
	}
	if cached != nil {
		call = call.IfNoneMatch(cached.Etag)
	}

	resp, err := call.Context(ctx).Do()
	if err != nil {
		if cached != nil && googleapi.IsNotModified(err) {
			var stored youtube.PlaylistItemListResponse
			if uerr := json.Unmarshal(cached.Payload, &stored); uerr != nil {
				return nil, fmt.Errorf("cache payload decode: %w", uerr)
			}
			return &stored, nil
		}
		return nil, fmt.Errorf("playlistItems.list: %w", err)
	}

	if !noCache && resp.Etag != "" {
		payload, merr := json.Marshal(resp)
		if merr == nil {
			_ = cache.Store(key, resp.Etag, payload)
		}
	}
	return resp, nil
}
