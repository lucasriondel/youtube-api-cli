package cmd

import "github.com/spf13/cobra"

func newRefCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "ref",
		Short: "Reference data: video categories, i18n languages, i18n regions",
	}
	c.AddCommand(newRefCategoriesCmd())
	c.AddCommand(newRefLanguagesCmd())
	c.AddCommand(newRefRegionsCmd())
	return c
}
