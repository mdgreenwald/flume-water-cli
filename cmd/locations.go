package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var locationsCmd = &cobra.Command{
	Use:   "locations",
	Short: "Manage Flume Water locations",
}

var locationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all locations",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, result, err := authenticate(cmd.Context())
		if err != nil {
			return err
		}

		locations, err := client.GetLocations(cmd.Context(), result.AccessToken, result.UserID, nil)
		if err != nil {
			return fmt.Errorf("failed to get locations: %w", err)
		}

		out := cmd.OutOrStdout()
		if len(locations) == 0 {
			_, _ = fmt.Fprintln(out, "No locations found")
			return nil
		}

		for i, loc := range locations {
			_, _ = fmt.Fprintf(out, "[%d] %s (ID: %s)\n", i+1, loc.Name, loc.ID.String())
			_, _ = fmt.Fprintf(out, "    Address: %s, %s, %s %s\n", loc.Address, loc.City, loc.State, loc.PostalCode)
			_, _ = fmt.Fprintf(out, "    Timezone: %s\n", loc.Timezone)
		}
		return nil
	},
}

func init() {
	locationsCmd.AddCommand(locationsListCmd)
	rootCmd.AddCommand(locationsCmd)
}
