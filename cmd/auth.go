package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/mdgreenwald/flume-water-cli/internal/cache"
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the Flume Water API and cache the token",
	Long: `Performs a live authentication against the Flume API using your credentials
and caches the resulting token for subsequent commands.

Run this command once to prime the cache. Subsequent commands (including cron
jobs) will use the cached token until it expires, at which point they will
re-authenticate automatically if credentials are available.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, result, err := authenticateFresh(cmd.Context())
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		_, _ = fmt.Fprintf(out, "Authentication successful\nUser ID: %s\n", result.UserID)

		// Show cache location and token expiry so users know what was written.
		if cacheDir, dirErr := cache.Dir(); dirErr == nil {
			_, _ = fmt.Fprintf(out, "Token cached: %s\n", filepath.Join(cacheDir, "token.json"))
		}
		if tc := cache.Load(); tc != nil {
			_, _ = fmt.Fprintf(out, "Token expires: %s\n", tc.ExpiresAt.UTC().Format(time.RFC3339))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
}
