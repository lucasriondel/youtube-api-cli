package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newPlaylistsUpdateCmd() *cobra.Command {
	var (
		title       string
		description string
		privacy     string
	)

	totalCost := ytapi.CostList + ytapi.CostUpdate

	c := &cobra.Command{
		Use:   "update <id> [--title <title>] [--description <text>] [--privacy private|unlisted|public]",
		Short: fmt.Sprintf("Update an existing playlist (cost: %d units)", totalCost),
		Long: fmt.Sprintf("Update an existing playlist's title, description, or privacy.\n\n"+
			"Quota cost: %d unit (playlists.list) + %d units (playlists.update) = %d units per call.\n"+
			"Only flags you pass are changed; other fields are preserved.\n"+
			"Use --dry-run to print the planned mutation without calling update.",
			ytapi.CostList, ytapi.CostUpdate, totalCost),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		id := args[0]

		titleSet := cmd.Flags().Changed("title")
		descSet := cmd.Flags().Changed("description")
		privacySet := cmd.Flags().Changed("privacy")

		if !titleSet && !descSet && !privacySet {
			return fmt.Errorf("at least one of --title, --description, --privacy must be provided")
		}

		if privacySet {
			switch privacy {
			case "private", "unlisted", "public":
			default:
				return fmt.Errorf("--privacy must be one of: private, unlisted, public (got %q)", privacy)
			}
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		resp, err := svc.Playlists.
			List([]string{"id", "snippet", "status"}).
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
		cur := resp.Items[0]

		newTitle, newDesc, newPrivacy := "", "", ""
		if cur.Snippet != nil {
			newTitle = cur.Snippet.Title
			newDesc = cur.Snippet.Description
		}
		if cur.Status != nil {
			newPrivacy = cur.Status.PrivacyStatus
		}
		if titleSet {
			newTitle = title
		}
		if descSet {
			newDesc = description
		}
		if privacySet {
			newPrivacy = privacy
		}

		if newTitle == "" {
			return fmt.Errorf("playlist title cannot be empty (YouTube API requires a non-empty title)")
		}

		pl := &youtube.Playlist{
			Id: id,
			Snippet: &youtube.PlaylistSnippet{
				Title:       newTitle,
				Description: newDesc,
			},
			Status: &youtube.PlaylistStatus{
				PrivacyStatus: newPrivacy,
			},
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostUpdate,
				"would update playlist id=%s title=%q privacy=%s description=%q",
				id, newTitle, newPrivacy, newDesc,
			)
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"FIELD", "VALUE"},
				[][]string{
					{"ID", id},
					{"TITLE", newTitle},
					{"PRIVACY", newPrivacy},
					{"DESCRIPTION", newDesc},
				},
				pl,
			)
		}

		updated, err := svc.Playlists.
			Update([]string{"snippet", "status"}, pl).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("playlists.update: %w", err)
		}

		t, p, d := "", "", ""
		if updated.Snippet != nil {
			t = updated.Snippet.Title
			d = updated.Snippet.Description
		}
		if updated.Status != nil {
			p = updated.Status.PrivacyStatus
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", updated.Id},
				{"TITLE", t},
				{"PRIVACY", p},
				{"DESCRIPTION", d},
			},
			updated,
		)
	}

	c.Flags().StringVar(&title, "title", "", "new playlist title")
	c.Flags().StringVar(&description, "description", "", "new playlist description")
	c.Flags().StringVar(&privacy, "privacy", "", "new privacy: private, unlisted, or public")
	return c
}
