package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/lucasrndl/yt/internal/ytapi"
)

func main() {
	ctx := context.Background()
	svc, err := ytapi.New(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("=== Channels.List(mine=true) ===")
	chResp, err := svc.Channels.List([]string{"id", "snippet", "contentDetails"}).Mine(true).Do()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, ch := range chResp.Items {
		fmt.Printf("  channel id=%s title=%q\n", ch.Id, ch.Snippet.Title)
		if ch.ContentDetails != nil && ch.ContentDetails.RelatedPlaylists != nil {
			rp := ch.ContentDetails.RelatedPlaylists
			fmt.Printf("    related: likes=%s uploads=%s\n", rp.Likes, rp.Uploads)
		}
	}

	fmt.Println("\n=== Playlists.List(mine=true) ===")
	plResp, err := svc.Playlists.List([]string{"id", "snippet", "status", "contentDetails"}).Mine(true).MaxResults(50).Do()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("  totalResults=%d returned=%d\n", plResp.PageInfo.TotalResults, len(plResp.Items))
	for _, p := range plResp.Items {
		fmt.Printf("  - %s  %q  (%d items, %s)\n", p.Id, p.Snippet.Title, p.ContentDetails.ItemCount, p.Status.PrivacyStatus)
	}

	for _, ch := range chResp.Items {
		fmt.Printf("\n=== Playlists.List(channelId=%s) ===\n", ch.Id)
		r, err := svc.Playlists.List([]string{"id", "snippet", "status", "contentDetails"}).ChannelId(ch.Id).MaxResults(50).Do()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		fmt.Printf("  totalResults=%d returned=%d\n", r.PageInfo.TotalResults, len(r.Items))
		for _, p := range r.Items {
			fmt.Printf("  - %s  %q  (%d items, %s)\n", p.Id, p.Snippet.Title, p.ContentDetails.ItemCount, p.Status.PrivacyStatus)
		}
	}

	fmt.Println("\n=== Liked playlist (LL) probe ===")
	llResp, err := svc.PlaylistItems.List([]string{"id", "snippet"}).PlaylistId("LL").MaxResults(5).Do()
	if err != nil {
		fmt.Fprintln(os.Stderr, "  error:", err)
	} else {
		fmt.Printf("  totalResults=%d sample=%d\n", llResp.PageInfo.TotalResults, len(llResp.Items))
		for _, it := range llResp.Items {
			fmt.Printf("  - %q\n", it.Snippet.Title)
		}
	}

	_ = json.NewEncoder(os.Stdout)
}
