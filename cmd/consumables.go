package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	flumewater "github.com/mdgreenwald/lib-flume-water"
	"github.com/spf13/cobra"
)

var (
	outputFlag      string
	pushGatewayFlag string
)

type consumableResult struct {
	DeviceID         string  `json:"device_id"`
	Name             string  `json:"name"`
	Installed        string  `json:"installed"`
	ExpiresGallons   int     `json:"expires_gallons"`
	UsedGallons      float64 `json:"used_gallons"`
	PercentUsed      float64 `json:"percent_used"`
	WarningThreshold int     `json:"warning_threshold"`
	Status           string  `json:"status"`
}

var consumablesCmd = &cobra.Command{
	Use:   "consumables",
	Short: "Track consumable water product life spans",
}

var consumablesStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show remaining life for configured consumables",
	RunE: func(cmd *cobra.Command, args []string) error {
		if outputFlag != "table" && outputFlag != "json" {
			return fmt.Errorf("unknown output format %q: use table or json", outputFlag)
		}
		if pushGatewayFlag != "" && cmd.Flags().Changed("output") {
			return fmt.Errorf("--push-gateway and --output are mutually exclusive")
		}

		if len(cfg.Consumables) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No consumables configured. Add consumables to ~/.config/flume/config.yaml")
			return nil
		}

		deviceIDs := make([]string, 0, len(cfg.Consumables))
		if deviceIDFlag != "" {
			if _, ok := cfg.Consumables[deviceIDFlag]; !ok {
				return fmt.Errorf("device %q has no consumables configured", deviceIDFlag)
			}
			deviceIDs = append(deviceIDs, deviceIDFlag)
		} else {
			for id := range cfg.Consumables {
				deviceIDs = append(deviceIDs, id)
			}
			sort.Strings(deviceIDs)
		}

		client, result, err := authenticate(cmd.Context())
		if err != nil {
			return err
		}

		var all []consumableResult
		for _, deviceID := range deviceIDs {
			consumables := cfg.Consumables[deviceID]

			earliest := time.Now()
			for _, c := range consumables {
				if c.Installed.Before(earliest) {
					earliest = c.Installed
				}
			}

			now := time.Now()
			queries := []flumewater.Query{
				{
					RequestID:     "consumables",
					Bucket:        "DAY",
					SinceDatetime: flumewater.Time{Time: earliest},
					UntilDatetime: flumewater.Time{Time: now},
				},
			}

			results, err := client.QueryDevice(cmd.Context(), result.AccessToken, result.UserID, deviceID, queries)
			if err != nil {
				return fmt.Errorf("failed to query device %s: %w", deviceID, err)
			}

			names := make([]string, 0, len(consumables))
			for name := range consumables {
				names = append(names, name)
			}
			sort.Strings(names)

			for _, name := range names {
				c := consumables[name]

				var used float64
				for _, r := range results {
					for _, dp := range r.Data {
						if !dp.Datetime.Before(c.Installed) {
							used += dp.Value
						}
					}
				}

				pct := used / float64(c.Expires) * 100

				threshold := cfg.Warning
				if c.Warning != nil {
					threshold = *c.Warning
				}

				status := "ok"
				switch {
				case pct >= 100:
					status = "expired"
				case pct >= float64(threshold):
					status = "warning"
				}

				all = append(all, consumableResult{
					DeviceID:         deviceID,
					Name:             name,
					Installed:        c.Installed.Format("2006-01-02"),
					ExpiresGallons:   c.Expires,
					UsedGallons:      math.Round(used*10) / 10,
					PercentUsed:      math.Round(pct*10) / 10,
					WarningThreshold: threshold,
					Status:           status,
				})
			}
		}

		if pushGatewayFlag != "" {
			return pushToGateway(cmd.Context(), pushGatewayFlag, prometheusText(all))
		}

		out := cmd.OutOrStdout()

		if outputFlag == "json" {
			data, err := json.MarshalIndent(all, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal JSON: %w", err)
			}
			_, _ = fmt.Fprintln(out, string(data))
			return nil
		}

		var lastDevice string
		for _, r := range all {
			if r.DeviceID != lastDevice {
				_, _ = fmt.Fprintf(out, "Device: %s\n", r.DeviceID)
				lastDevice = r.DeviceID
			}
			tableStatus := ""
			switch r.Status {
			case "warning":
				tableStatus = "  WARNING: approaching end of life"
			case "expired":
				tableStatus = "  EXPIRED"
			}
			_, _ = fmt.Fprintf(out,
				"  %-14s installed: %s  %7.1f / %d gal (%.1f%%)%s\n",
				r.Name,
				r.Installed,
				r.UsedGallons,
				r.ExpiresGallons,
				r.PercentUsed,
				tableStatus,
			)
		}
		return nil
	},
}

// prometheusText serializes results as Prometheus text exposition format.
func prometheusText(results []consumableResult) string {
	type metric struct {
		name  string
		help  string
		value func(consumableResult) float64
	}
	metrics := []metric{
		{"flume_consumable_percent_used", "Percentage of consumable life used",
			func(r consumableResult) float64 { return r.PercentUsed }},
		{"flume_consumable_used_gallons", "Gallons consumed since installation",
			func(r consumableResult) float64 { return r.UsedGallons }},
		{"flume_consumable_expires_gallons", "Total gallon capacity of consumable",
			func(r consumableResult) float64 { return float64(r.ExpiresGallons) }},
		{"flume_consumable_warning_threshold", "Warning threshold percentage",
			func(r consumableResult) float64 { return float64(r.WarningThreshold) }},
	}
	var b strings.Builder
	for _, m := range metrics {
		fmt.Fprintf(&b, "# HELP %s %s\n", m.name, m.help)
		fmt.Fprintf(&b, "# TYPE %s gauge\n", m.name)
		for _, r := range results {
			fmt.Fprintf(&b, "%s{device_id=%q,name=%q} %g\n", m.name, r.DeviceID, r.Name, m.value(r))
		}
	}
	return b.String()
}

// pushToGateway sends metrics to a Prometheus Pushgateway via HTTP PUT.
func pushToGateway(ctx context.Context, gatewayURL, body string) error {
	url := strings.TrimRight(gatewayURL, "/") + "/metrics/job/flume_consumables"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("build push request: %w", err)
	}
	req.Header.Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("push to gateway: %w", err)
	}
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()

	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("gateway returned %s", resp.Status)
	}
	return nil
}

func init() {
	consumablesStatusCmd.Flags().StringVar(&deviceIDFlag, "device-id", "", "show consumables for a specific device only")
	consumablesStatusCmd.Flags().StringVarP(&outputFlag, "output", "o", "table", "output format (table, json)")
	consumablesStatusCmd.Flags().StringVar(&pushGatewayFlag, "push-gateway", "", "Prometheus Pushgateway URL (e.g. http://localhost:9091)")
	consumablesCmd.AddCommand(consumablesStatusCmd)
	rootCmd.AddCommand(consumablesCmd)
}
