package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mdgreenwald/flume-water-cli/internal/cache"
	"github.com/mdgreenwald/flume-water-cli/internal/config"
	flumewater "github.com/mdgreenwald/lib-flume-water"
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

var (
	envFileFlag      string
	clientIDFlag     string
	clientSecretFlag string
	emailFlag        string
	passwordFlag     string
)

// cfg holds the loaded user configuration. Available to all commands after
// PersistentPreRunE fires.
var cfg *config.Config

// newClient is a factory that tests can override to inject a custom client.
var newClient = func() *flumewater.Client {
	return flumewater.NewClient()
}

var rootCmd = &cobra.Command{
	Use:     "flume",
	Short:   "CLI tool for interacting with the Flume Water API",
	Version: version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		return err
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&envFileFlag, "env-file", "", "path to .env file with credentials")
	rootCmd.PersistentFlags().StringVar(&clientIDFlag, "client-id", "", "Flume API client ID (overrides FLUME_CLIENT_ID)")
	rootCmd.PersistentFlags().StringVar(&clientSecretFlag, "client-secret", "", "Flume API client secret (overrides FLUME_CLIENT_SECRET)")
	rootCmd.PersistentFlags().StringVar(&emailFlag, "email", "", "Flume account email (overrides FLUME_USER_EMAIL)")
	rootCmd.PersistentFlags().StringVar(&passwordFlag, "password", "", "Flume account password (overrides FLUME_USER_PASSWORD)")
}

// authenticate checks the on-disk token cache first. If no valid cached token
// is found it falls through to a live API call and updates the cache.
func authenticate(ctx context.Context) (*flumewater.Client, *flumewater.AuthResult, error) {
	client := newClient()

	if tc := cache.Load(); tc != nil && tc.IsValid() {
		return client, tc.ToAuthResult(), nil
	}

	result, err := doFreshAuth(ctx, client)
	if err != nil {
		return nil, nil, err
	}
	return client, result, nil
}

// authenticateFresh always performs a live API call regardless of any cached
// token. It is used by the auth command to verify that credentials still work.
// On success the cache is updated.
func authenticateFresh(ctx context.Context) (*flumewater.Client, *flumewater.AuthResult, error) {
	client := newClient()
	result, err := doFreshAuth(ctx, client)
	if err != nil {
		return nil, nil, err
	}
	return client, result, nil
}

// doFreshAuth resolves credentials, authenticates against the API, and
// persists the result to the token cache.
func doFreshAuth(ctx context.Context, client *flumewater.Client) (*flumewater.AuthResult, error) {
	cid, cs, em, pw, err := resolveCredentials()
	if err != nil {
		return nil, err
	}

	result, err := client.Authenticate(ctx, cid, cs, em, pw)
	if err != nil {
		return nil, err
	}

	if saveErr := cache.Save(result); saveErr != nil {
		// Cache failures are non-fatal — the token is still usable this session.
		_, _ = fmt.Fprintf(os.Stderr, "warning: failed to save token cache: %v\n", saveErr)
	}
	return result, nil
}

// resolveCredentials returns credentials by checking (in priority order):
// CLI flags, FLUME_* environment variables, then a .env file.
func resolveCredentials() (cid, cs, em, pw string, err error) {
	cid = coalesce(clientIDFlag, os.Getenv("FLUME_CLIENT_ID"))
	cs = coalesce(clientSecretFlag, os.Getenv("FLUME_CLIENT_SECRET"))
	em = coalesce(emailFlag, os.Getenv("FLUME_USER_EMAIL"))
	pw = coalesce(passwordFlag, os.Getenv("FLUME_USER_PASSWORD"))

	if cid != "" && cs != "" && em != "" && pw != "" {
		return
	}

	envPath := envFileFlag
	if envPath == "" {
		if _, statErr := os.Stat(".env"); statErr == nil {
			envPath = ".env"
		}
	}

	if envPath == "" {
		return "", "", "", "", fmt.Errorf(
			"credentials required: set FLUME_CLIENT_ID, FLUME_CLIENT_SECRET, " +
				"FLUME_USER_EMAIL, FLUME_USER_PASSWORD or use --env-file",
		)
	}

	creds, loadErr := flumewater.LoadCredentialsFromEnv(envPath)
	if loadErr != nil {
		return "", "", "", "", fmt.Errorf("failed to load credentials from %s: %w", envPath, loadErr)
	}

	// Flags and env vars take precedence over the file.
	cid = coalesce(cid, creds.ClientID)
	cs = coalesce(cs, creds.ClientSecret)
	em = coalesce(em, creds.UserEmail)
	pw = coalesce(pw, creds.UserPassword)
	return
}

func coalesce(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
