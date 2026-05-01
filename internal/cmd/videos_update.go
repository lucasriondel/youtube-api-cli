package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newVideosUpdateCmd() *cobra.Command {
	var (
		title       string
		description string
		tagsCSV     string
		category    string
	)

	totalCost := ytapi.CostList + ytapi.CostUpdate

	c := &cobra.Command{
		Use:   "update <id-or-url> [--title <title>] [--description <text>] [--tags <csv>] [--category <id>]",
		Short: fmt.Sprintf("Update an existing video's metadata (cost: %d units)", totalCost),
		Long: fmt.Sprintf("Update title, description, tags, or category on a video you own.\n\n"+
			"The argument may be a raw video id (e.g. dQw4w9WgXcQ) or a YouTube URL\n"+
			"(watch?v=..., youtu.be/..., shorts/...).\n\n"+
			"Quota cost: %d unit (videos.list) + %d units (videos.update) = %d units per call.\n"+
			"Patch semantics: only flags you pass are changed; other snippet fields are\n"+
			"preserved. The YouTube API requires the categoryId on every videos.update,\n"+
			"so we always read the current snippet first to keep it intact.\n\n"+
			"--tags takes a comma-separated list (e.g. --tags=\"go,cli,youtube\").\n"+
			"Pass --tags=\"\" to clear all tags.\n"+
			"Use --dry-run to print the planned mutation without calling update.\n\n"+
			"Note: you can only update videos owned by the authenticated account.",
			ytapi.CostList, ytapi.CostUpdate, totalCost),
		Args: cobra.ExactArgs(1),
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		id, err := parseVideoID(args[0])
		if err != nil {
			return fmt.Errorf("invalid video reference %q: %w", args[0], err)
		}

		titleSet := cmd.Flags().Changed("title")
		descSet := cmd.Flags().Changed("description")
		tagsSet := cmd.Flags().Changed("tags")
		categorySet := cmd.Flags().Changed("category")

		if !titleSet && !descSet && !tagsSet && !categorySet {
			return fmt.Errorf("at least one of --title, --description, --tags, --category must be provided")
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		resp, err := svc.Videos.
			List([]string{"id", "snippet"}).
			Id(id).
			MaxResults(1).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("videos.list: %w", err)
		}
		if len(resp.Items) == 0 {
			return fmt.Errorf("video %q not found", id)
		}
		cur := resp.Items[0]
		if cur.Snippet == nil {
			return fmt.Errorf("video %q has no snippet (cannot update)", id)
		}

		newTitle := cur.Snippet.Title
		newDesc := cur.Snippet.Description
		newTags := cur.Snippet.Tags
		newCategory := cur.Snippet.CategoryId

		if titleSet {
			newTitle = title
		}
		if descSet {
			newDesc = description
		}
		if tagsSet {
			newTags = parseTagsCSV(tagsCSV)
		}
		if categorySet {
			newCategory = category
		}

		if strings.TrimSpace(newTitle) == "" {
			return fmt.Errorf("video title cannot be empty (YouTube API requires a non-empty title)")
		}
		if strings.TrimSpace(newCategory) == "" {
			return fmt.Errorf("video categoryId cannot be empty (YouTube API requires a categoryId on update)")
		}

		v := &youtube.Video{
			Id: id,
			Snippet: &youtube.VideoSnippet{
				Title:       newTitle,
				Description: newDesc,
				Tags:        newTags,
				CategoryId:  newCategory,
			},
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostUpdate,
				"would update video id=%s title=%q category=%s",
				id, newTitle, newCategory,
			)
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"FIELD", "VALUE"},
				[][]string{
					{"ID", id},
					{"TITLE", newTitle},
					{"CATEGORY", newCategory},
					{"TAGS", strings.Join(newTags, ", ")},
					{"DESCRIPTION", newDesc},
				},
				v,
			)
		}

		updated, err := svc.Videos.
			Update([]string{"snippet"}, v).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("videos.update: %w", err)
		}

		t, d, cat := "", "", ""
		var tags []string
		if updated.Snippet != nil {
			t = updated.Snippet.Title
			d = updated.Snippet.Description
			tags = updated.Snippet.Tags
			cat = updated.Snippet.CategoryId
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", updated.Id},
				{"TITLE", t},
				{"CATEGORY", cat},
				{"TAGS", strings.Join(tags, ", ")},
				{"DESCRIPTION", d},
			},
			updated,
		)
	}

	c.Flags().StringVar(&title, "title", "", "new video title")
	c.Flags().StringVar(&description, "description", "", "new video description")
	c.Flags().StringVar(&tagsCSV, "tags", "", "new tags as a comma-separated list (empty string clears all tags)")
	c.Flags().StringVar(&category, "category", "", "new YouTube video category id (see videoCategories.list)")
	return c
}

// parseTagsCSV splits a comma-separated tag string into a trimmed, non-empty slice.
// "" returns nil (clears all tags); "go, cli ,, youtube" returns ["go","cli","youtube"].
func parseTagsCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
