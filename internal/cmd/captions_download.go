package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/lucasrndl/yt/internal/ytapi"
	"github.com/spf13/cobra"
)

func newCaptionsDownloadCmd() *cobra.Command {
	var (
		format string
		out    string
	)

	c := &cobra.Command{
		Use:   "download <caption-id>",
		Short: fmt.Sprintf("Download a caption track (cost: %d units)", ytapi.CostCaptionsDownload),
		Long: fmt.Sprintf("Download a caption track via captions.download.\n\n"+
			"Cost: %d units per call. <caption-id> is the CAPTION_ID column from\n"+
			"`yt captions list`.\n\n"+
			"--format selects the wire format: sbv, srt (default), or vtt. The API\n"+
			"converts the stored track on the fly.\n\n"+
			"-o writes the body to a file; otherwise the body is written to stdout\n"+
			"(no trailing newline added). The global --json and --plain flags do\n"+
			"not apply — caption output is the raw subtitle file.\n\n"+
			"Note: the API only allows downloading tracks owned by the authenticated\n"+
			"channel. Downloading another channel's tracks (including auto-generated\n"+
			"ASR) returns 403 forbidden.",
			ytapi.CostCaptionsDownload),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(args[0])
			if id == "" {
				return fmt.Errorf("invalid caption id: empty value")
			}
			if err := validateCaptionFormat(format); err != nil {
				return err
			}

			ctx := cmd.Context()
			svc, err := ytapi.New(ctx)
			if err != nil {
				return err
			}

			call := svc.Captions.Download(id).Tfmt(format).Context(ctx)
			resp, err := call.Download()
			if err != nil {
				return fmt.Errorf("captions.download: %w", err)
			}
			defer resp.Body.Close()

			var w io.Writer = cmd.OutOrStdout()
			if out != "" {
				f, err := os.OpenFile(out, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
				if err != nil {
					return fmt.Errorf("open output file: %w", err)
				}
				defer f.Close()
				w = f
			}

			if _, err := io.Copy(w, resp.Body); err != nil {
				return fmt.Errorf("read caption body: %w", err)
			}
			return nil
		},
	}

	c.Flags().StringVar(&format, "format", "srt", "subtitle format: sbv, srt, or vtt")
	c.Flags().StringVarP(&out, "output", "o", "", "write the caption body to this file (default: stdout)")
	return c
}

func validateCaptionFormat(f string) error {
	switch f {
	case "sbv", "srt", "vtt":
		return nil
	default:
		return fmt.Errorf("invalid --format %q (want one of: sbv, srt, vtt)", f)
	}
}
