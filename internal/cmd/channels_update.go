package cmd

import (
	"fmt"
	"strings"

	"github.com/lucasrndl/yt/internal/output"
	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
	"google.golang.org/api/youtube/v3"
)

func newChannelsUpdateCmd() *cobra.Command {
	var (
		description string
		keywords    string
		country     string
	)

	totalCost := ytapi.CostList + ytapi.CostUpdate

	c := &cobra.Command{
		Use:   "update [--description <text>] [--keywords <csv>] [--country <code>]",
		Short: fmt.Sprintf("Update branding settings on the authenticated channel (cost: %d units)", totalCost),
		Long: fmt.Sprintf("Update brandingSettings on the authenticated user's own channel.\n\n"+
			"Quota cost: %d unit (channels.list mine=true) + %d units (channels.update) = %d units per call.\n"+
			"Patch semantics: only flags you pass are changed; other branding fields are\n"+
			"preserved. The YouTube API requires the channel title on every channels.update,\n"+
			"so we always read the current branding first to keep it intact.\n\n"+
			"--keywords is a free-form string (the API stores it as-is, comma- or space-\n"+
			"separated keywords are conventional). Pass --keywords=\"\" to clear keywords.\n"+
			"--country is a two-letter ISO 3166-1 country code (e.g. US, FR).\n"+
			"Pass --country=\"\" to clear the country.\n\n"+
			"Use --dry-run to print the planned mutation without calling update.\n\n"+
			"Note: only the authenticated user's own channel can be updated.",
			ytapi.CostList, ytapi.CostUpdate, totalCost),
		Args: cobra.NoArgs,
	}

	dryRun := addDryRunFlag(c)
	c.RunE = func(cmd *cobra.Command, args []string) error {
		descSet := cmd.Flags().Changed("description")
		keywordsSet := cmd.Flags().Changed("keywords")
		countrySet := cmd.Flags().Changed("country")

		if !descSet && !keywordsSet && !countrySet {
			return fmt.Errorf("at least one of --description, --keywords, --country must be provided")
		}

		ctx := cmd.Context()
		svc, err := ytapi.New(ctx)
		if err != nil {
			return err
		}

		resp, err := svc.Channels.
			List([]string{"id", "brandingSettings"}).
			Mine(true).
			MaxResults(1).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("channels.list: %w", err)
		}
		if len(resp.Items) == 0 {
			return fmt.Errorf("no channel found for the authenticated user")
		}
		cur := resp.Items[0]
		if cur.BrandingSettings == nil || cur.BrandingSettings.Channel == nil {
			return fmt.Errorf("channel %q has no brandingSettings.channel (cannot update)", cur.Id)
		}

		curChannel := cur.BrandingSettings.Channel
		newTitle := curChannel.Title
		newDesc := curChannel.Description
		newKeywords := curChannel.Keywords
		newCountry := curChannel.Country

		if descSet {
			newDesc = description
		}
		if keywordsSet {
			newKeywords = keywords
		}
		if countrySet {
			newCountry = country
		}

		if strings.TrimSpace(newTitle) == "" {
			return fmt.Errorf("channel title cannot be empty (YouTube API requires a non-empty title on update)")
		}

		settings := &youtube.ChannelSettings{
			Title:       newTitle,
			Description: newDesc,
			Keywords:    newKeywords,
			Country:     newCountry,
		}
		if descSet && newDesc == "" {
			settings.ForceSendFields = append(settings.ForceSendFields, "Description")
		}
		if keywordsSet && newKeywords == "" {
			settings.ForceSendFields = append(settings.ForceSendFields, "Keywords")
		}
		if countrySet && newCountry == "" {
			settings.ForceSendFields = append(settings.ForceSendFields, "Country")
		}

		ch := &youtube.Channel{
			Id: cur.Id,
			BrandingSettings: &youtube.ChannelBrandingSettings{
				Channel: settings,
			},
		}

		if *dryRun {
			printDryRun(cmd.OutOrStderr(), ytapi.CostUpdate,
				"would update channel id=%s title=%q country=%q",
				cur.Id, newTitle, newCountry,
			)
			format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
			return output.Render(
				cmd.OutOrStdout(),
				format,
				[]string{"FIELD", "VALUE"},
				[][]string{
					{"ID", cur.Id},
					{"TITLE", newTitle},
					{"COUNTRY", newCountry},
					{"KEYWORDS", newKeywords},
					{"DESCRIPTION", newDesc},
				},
				ch,
			)
		}

		updated, err := svc.Channels.
			Update([]string{"brandingSettings"}, ch).
			Context(ctx).
			Do()
		if err != nil {
			return fmt.Errorf("channels.update: %w", err)
		}

		t, d, k, ctry := "", "", "", ""
		if updated.BrandingSettings != nil && updated.BrandingSettings.Channel != nil {
			t = updated.BrandingSettings.Channel.Title
			d = updated.BrandingSettings.Channel.Description
			k = updated.BrandingSettings.Channel.Keywords
			ctry = updated.BrandingSettings.Channel.Country
		}

		format := output.FormatFromFlags(Globals.JSON, Globals.Plain)
		return output.Render(
			cmd.OutOrStdout(),
			format,
			[]string{"FIELD", "VALUE"},
			[][]string{
				{"ID", updated.Id},
				{"TITLE", t},
				{"COUNTRY", ctry},
				{"KEYWORDS", k},
				{"DESCRIPTION", d},
			},
			updated,
		)
	}

	c.Flags().StringVar(&description, "description", "", "new channel description (empty string clears it)")
	c.Flags().StringVar(&keywords, "keywords", "", "channel keywords (free-form string; empty string clears them)")
	c.Flags().StringVar(&country, "country", "", "two-letter ISO 3166-1 country code (empty string clears it)")
	return c
}
