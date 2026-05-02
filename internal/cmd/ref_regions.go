package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newRefRegionsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "regions",
		Short: fmt.Sprintf("List YouTube i18n regions (cost: %d unit)", ytapi.CostList),
		Long: fmt.Sprintf("List the regions where YouTube is available via i18nRegions.list.\n\n"+
			"Cost: %d unit per call (returns the full list — no pagination).\n\n"+
			"The GL column is the 2-letter ISO 3166-1 alpha-2 region code accepted\n"+
			"as --region by `yt ref categories` and other endpoints that scope\n"+
			"results geographically.",
			ytapi.CostList),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			resp, err := svc.I18nRegions.List([]string{"id", "snippet"}).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("i18nRegions.list: %w", err)
			}

			rows := make([][]string, 0, len(resp.Items))
			for _, reg := range resp.Items {
				rows = append(rows, []string{
					reg.Id,
					i18nRegionGl(reg),
					i18nRegionName(reg),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"REGION_ID", "GL", "NAME"},
				rows,
				resp.Items,
			)
		},
	}

	return c
}

func i18nRegionGl(r *youtube.I18nRegion) string {
	if r.Snippet == nil {
		return ""
	}
	return r.Snippet.Gl
}

func i18nRegionName(r *youtube.I18nRegion) string {
	if r.Snippet == nil {
		return ""
	}
	return r.Snippet.Name
}
