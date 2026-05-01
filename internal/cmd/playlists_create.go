package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newPlaylistsCreateCmd() *cobra.Command {
	var (
		title       string
		description string
		privacy     string
	)

	c := &cobra.Command{
		Use:   "create --title <title> [--description <text>] [--privacy private|unlisted|public]",
		Short: fmt.Sprintf("Create a new playlist (cost: %d units)", ytapi.CostInsert),
		Long: fmt.Sprintf("Create a new playlist on the authenticated user's account.\n\n"+
			"Quota cost: %d units per call.\n"+
			"Use --dry-run to print the planned mutation without calling the API.", ytapi.CostInsert),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		if title == "" {
			return fmt.Errorf("--title is required")
		}
		switch privacy {
		case "private", "unlisted", "public":
		default:
			return fmt.Errorf("--privacy must be one of: private, unlisted, public (got %q)", privacy)
		}

		pl := &youtube.Playlist{
			Snippet: &youtube.PlaylistSnippet{
				Title:       title,
				Description: description,
			},
			Status: &youtube.PlaylistStatus{
				PrivacyStatus: privacy,
			},
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostInsert,
				"would create playlist title=%q privacy=%s description=%q",
				title, privacy, description,
			)
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"FIELD", "VALUE"},
				[][]string{
					{"TITLE", title},
					{"PRIVACY", privacy},
					{"DESCRIPTION", description},
				},
				pl,
			)
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		created, err := svc.Playlists.
			Insert([]string{"snippet", "status"}, pl).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("playlists.insert: %w", err)
		}

		t, p, d := "", "", ""
		if created.Snippet != nil {
			t = created.Snippet.Title
			d = created.Snippet.Description
		}
		if created.Status != nil {
			p = created.Status.PrivacyStatus
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", created.Id},
				{"TITLE", t},
				{"PRIVACY", p},
				{"DESCRIPTION", d},
			},
			created,
		)
	}

	c.Flags().StringVar(&title, "title", "", "playlist title (required)")
	c.Flags().StringVar(&description, "description", "", "playlist description")
	c.Flags().StringVar(&privacy, "privacy", "private", "privacy: private, unlisted, or public")
	return c
}
