package cmd

import "github.com/spf13/cobra"

// Flags available on every command for scriptable output.
type GlobalFlags struct {
	JSON  bool
	Plain bool
}

var Globals GlobalFlags

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "yt",
		Short:         "CLI for managing your YouTube account",
		Long:          "yt is an agent-friendly CLI for inspecting and reorganizing your YouTube playlists via the YouTube Data API v3.",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().BoolVar(&Globals.JSON, "json", false, "output JSON")
	root.PersistentFlags().BoolVar(&Globals.Plain, "plain", false, "output tab-separated values (no header)")

	root.AddCommand(newAuthCmd())
	root.AddCommand(newPlaylistsCmd())
	root.AddCommand(newItemsCmd())
	root.AddCommand(newVideosCmd())
	return root
}
