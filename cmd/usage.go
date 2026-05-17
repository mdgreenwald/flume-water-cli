package cmd

import (
	"fmt"
	"time"

	flumewater "github.com/mdgreenwald/lib-flume-water"
	"github.com/spf13/cobra"
)

var (
	deviceIDFlag string
	daysFlag     int
	bucketFlag   string
)

var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Query water usage data",
}

var usageQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query water usage for a device",
	RunE: func(cmd *cobra.Command, args []string) error {
		if deviceIDFlag == "" && cfg != nil && cfg.DefaultDeviceID != "" {
			deviceIDFlag = cfg.DefaultDeviceID
		}
		if deviceIDFlag == "" {
			return fmt.Errorf("--device-id is required")
		}

		client, result, err := authenticate(cmd.Context())
		if err != nil {
			return err
		}

		now := time.Now()
		since := now.AddDate(0, 0, -daysFlag)

		queries := []flumewater.Query{
			{
				RequestID:     "usage",
				Bucket:        bucketFlag,
				SinceDatetime: flumewater.Time{Time: since},
				UntilDatetime: flumewater.Time{Time: now},
			},
		}

		results, err := client.QueryDevice(cmd.Context(), result.AccessToken, result.UserID, deviceIDFlag, queries)
		if err != nil {
			return fmt.Errorf("failed to query device: %w", err)
		}

		out := cmd.OutOrStdout()
		for _, r := range results {
			if len(r.Data) == 0 {
				_, _ = fmt.Fprintln(out, "No data found for the specified period")
				continue
			}
			_, _ = fmt.Fprintf(out, "Usage data (%d data points):\n", len(r.Data))
			for _, dp := range r.Data {
				date := dp.Datetime.Format("2006-01-02")
				_, _ = fmt.Fprintf(out, "  %s: %.2f gallons\n", date, dp.Value)
			}
		}
		return nil
	},
}

func init() {
	usageQueryCmd.Flags().StringVar(&deviceIDFlag, "device-id", "", "device ID to query (required)")
	usageQueryCmd.Flags().IntVar(&daysFlag, "days", 30, "number of days to query")
	usageQueryCmd.Flags().StringVar(&bucketFlag, "bucket", "DAY", "time bucket (MIN, HOUR, DAY, WEEK, MON, YR)")
	usageCmd.AddCommand(usageQueryCmd)
	rootCmd.AddCommand(usageCmd)
}
