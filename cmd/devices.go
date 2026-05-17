package cmd

import (
	"fmt"

	flumewater "github.com/mdgreenwald/lib-flume-water"
	"github.com/spf13/cobra"
)

var locationIDFlag string

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "Manage Flume Water devices",
}

var devicesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, result, err := authenticate(cmd.Context())
		if err != nil {
			return err
		}

		var params *flumewater.DeviceListParams
		if locationIDFlag != "" {
			p := flumewater.DefaultDeviceListParams()
			p.LocationID = locationIDFlag
			params = &p
		}

		devices, err := client.GetDevices(cmd.Context(), result.AccessToken, result.UserID, params)
		if err != nil {
			return fmt.Errorf("failed to get devices: %w", err)
		}

		out := cmd.OutOrStdout()
		if len(devices) == 0 {
			_, _ = fmt.Fprintln(out, "No devices found")
			return nil
		}

		for i, dev := range devices {
			locID := "N/A"
			if dev.Location != nil {
				locID = dev.Location.ID.String()
			}
			_, _ = fmt.Fprintf(out, "[%d] %s (ID: %s)\n", i+1, dev.Type.String(), dev.ID.String())
			_, _ = fmt.Fprintf(out, "    Location ID: %s\n", locID)
			_, _ = fmt.Fprintf(out, "    Last Seen: %s\n", dev.LastSeen)
		}
		return nil
	},
}

func init() {
	devicesListCmd.Flags().StringVar(&locationIDFlag, "location-id", "", "filter devices by location ID")
	devicesCmd.AddCommand(devicesListCmd)
	rootCmd.AddCommand(devicesCmd)
}
