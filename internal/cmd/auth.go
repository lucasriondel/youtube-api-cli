package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/lucasrndl/yt/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2/google"
)

func newAuthCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication with YouTube",
	}
	c.AddCommand(newAuthCredentialsCmd())
	c.AddCommand(newAuthLoginCmd())
	c.AddCommand(newAuthStatusCmd())
	c.AddCommand(newAuthLogoutCmd())
	return c
}

func newAuthCredentialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credentials <path-to-client_secret.json>",
		Short: "Store OAuth2 client credentials downloaded from Google Cloud Console",
		Long: "Reads the JSON file you downloaded from Google Cloud Console (Desktop app OAuth client) " +
			"and stores its contents securely in the OS keyring. After this, run `yt auth login`.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			raw, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("read %s: %w", args[0], err)
			}
			if _, err := google.ConfigFromJSON(raw, auth.DefaultScopes()...); err != nil {
				return fmt.Errorf("invalid client secret JSON: %w", err)
			}
			if err := auth.SaveClientSecret(raw); err != nil {
				return fmt.Errorf("save credentials: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Credentials stored. Next: run `yt auth login`.")
			return nil
		},
	}
}

func newAuthLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authorize the CLI against your YouTube account",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := auth.LoadConfig(auth.DefaultScopes())
			if err != nil {
				return err
			}
			tok, err := auth.Login(cmd.Context(), cfg)
			if err != nil {
				return err
			}
			if err := auth.SaveToken(tok); err != nil {
				return fmt.Errorf("save token: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged in.")
			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show whether credentials and a token are stored",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()

			if _, err := auth.LoadClientSecret(); err != nil {
				if errors.Is(err, auth.ErrNoClientSecret) {
					fmt.Fprintln(out, "No client credentials stored. Run `yt auth credentials <path>`.")
					return nil
				}
				return err
			}
			fmt.Fprintln(out, "Client credentials: stored.")

			tok, err := auth.LoadToken()
			if err != nil {
				if errors.Is(err, auth.ErrNoToken) {
					fmt.Fprintln(out, "Not logged in. Run `yt auth login`.")
					return nil
				}
				return err
			}
			if tok.Valid() {
				fmt.Fprintf(out, "Logged in. Access token expires at %s.\n", tok.Expiry.Format("2006-01-02 15:04:05 MST"))
			} else {
				fmt.Fprintf(out, "Logged in. Access token expired at %s — will refresh on next call.\n", tok.Expiry.Format("2006-01-02 15:04:05 MST"))
			}
			return nil
		},
	}
}

func newAuthLogoutCmd() *cobra.Command {
	var all bool
	c := &cobra.Command{
		Use:   "logout",
		Short: "Delete the stored token (use --all to also remove client credentials)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.DeleteToken(); err != nil {
				return err
			}
			if all {
				if err := auth.DeleteClientSecret(); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Logged out and removed client credentials.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
	c.Flags().BoolVar(&all, "all", false, "also remove stored client credentials")
	return c
}
