package cmd

import (
	"fmt"
	"strconv"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newRefCategoriesCmd() *cobra.Command {
	var region string

	c := &cobra.Command{
		Use:   "categories",
		Short: fmt.Sprintf("List YouTube video categories (cost: %d unit)", ytapi.CostList),
		Long: fmt.Sprintf("List the video categories available in a region via videoCategories.list.\n\n"+
			"Cost: %d unit per call (returns the full list — no pagination).\n\n"+
			"--region accepts a 2-letter ISO country code (default US). The set of\n"+
			"categories and which ones are 'assignable' (i.e. usable when uploading\n"+
			"or updating a video) varies per region. The CATEGORY_ID column is the\n"+
			"value to feed to `yt videos update --category <id>`.",
			ytapi.CostList),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if region == "" {
				return fmt.Errorf("--region cannot be empty")
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			resp, err := svc.VideoCategories.List([]string{"id", "snippet"}).
				RegionCode(region).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("videoCategories.list: %w", err)
			}

			rows := make([][]string, 0, len(resp.Items))
			for _, cat := range resp.Items {
				rows = append(rows, []string{
					cat.Id,
					videoCategoryTitle(cat),
					strconv.FormatBool(videoCategoryAssignable(cat)),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"CATEGORY_ID", "TITLE", "ASSIGNABLE"},
				rows,
				resp.Items,
			)
		},
	}

	c.Flags().StringVar(&region, "region", "US", "ISO 3166-1 alpha-2 region code (e.g. US, FR, JP)")
	return c
}

func videoCategoryTitle(c *youtube.VideoCategory) string {
	if c.Snippet == nil {
		return ""
	}
	return c.Snippet.Title
}

func videoCategoryAssignable(c *youtube.VideoCategory) bool {
	if c.Snippet == nil {
		return false
	}
	return c.Snippet.Assignable
}
