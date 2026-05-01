package cmd

import (
	"fmt"

	"github.com/lucasrndl/yt/internal/cache"
	"github.com/spf13/cobra"
)

func newCacheCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "cache",
		Short: "Manage the on-disk response cache",
		Long: "yt caches API list responses by etag under os.UserCacheDir()/yt/. " +
			"Cached entries are revalidated via If-None-Match on every refetch, so they " +
			"cost 0 quota units when unchanged. Use `yt cache clear` to wipe them.",
	}
	c.AddCommand(newCacheClearCmd())
	c.AddCommand(newCacheInfoCmd())
	return c
}

func newCacheClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Delete every cached API response",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := cache.Clear()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %d cache entr%s.\n", n, plural(n, "y", "ies"))
			return nil
		},
	}
}

func newCacheInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show the cache directory path",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := cache.Dir()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), dir)
			return nil
		},
	}
}

func plural(n int, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
