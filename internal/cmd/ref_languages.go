package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newRefLanguagesCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "languages",
		Short: fmt.Sprintf("List YouTube i18n languages (cost: %d unit)", ytapi.CostList),
		Long: fmt.Sprintf("List the application languages YouTube supports via i18nLanguages.list.\n\n"+
			"Cost: %d unit per call (returns the full list — no pagination).\n\n"+
			"The HL column is the BCP-47 code accepted by API parameters such as\n"+
			"--hl on the videoCategories endpoint. NAME is the language's name in\n"+
			"the language itself.",
			ytapi.CostList),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			resp, err := svc.I18nLanguages.List([]string{"id", "snippet"}).
				Context(ctx).
				Do()
			if err != nil {
				return fmt.Errorf("i18nLanguages.list: %w", err)
			}

			rows := make([][]string, 0, len(resp.Items))
			for _, lang := range resp.Items {
				rows = append(rows, []string{
					lang.Id,
					i18nLanguageHl(lang),
					i18nLanguageName(lang),
				})
			}

			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"LANGUAGE_ID", "HL", "NAME"},
				rows,
				resp.Items,
			)
		},
	}

	return c
}

func i18nLanguageHl(l *youtube.I18nLanguage) string {
	if l.Snippet == nil {
		return ""
	}
	return l.Snippet.Hl
}

func i18nLanguageName(l *youtube.I18nLanguage) string {
	if l.Snippet == nil {
		return ""
	}
	return l.Snippet.Name
}
