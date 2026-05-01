package cmd

import "github.com/spf13/cobra"

func newPlaylistsCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "playlists",
		Short: "Inspect and manage your playlists",
	}
	c.AddCommand(newPlaylistsListCmd())
	c.AddCommand(newPlaylistsShowCmd())
	c.AddCommand(newPlaylistsCreateCmd())
	c.AddCommand(newPlaylistsUpdateCmd())
	c.AddCommand(newPlaylistsDeleteCmd())
	return c
}
