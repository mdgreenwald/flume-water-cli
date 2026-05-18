package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mdgreenwald/flume-water-cli/internal/config"
)

// mockInstalled is the install date that includes all mock query data points
// (2026-01-01 and 2026-01-02). Total mock usage = 20.8 gal.
var mockInstalled = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func TestConsumablesStatus(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	out, err := runCmd(t, "consumables", "status")
	if err != nil {
		t.Fatalf("consumables status failed: %v", err)
	}

	if !strings.Contains(out, "Device: dev1") {
		t.Errorf("expected device header, got: %q", out)
	}
	if !strings.Contains(out, "charcoal") {
		t.Errorf("expected filter name in output, got: %q", out)
	}
	if !strings.Contains(out, "gal") {
		t.Errorf("expected gallon unit in output, got: %q", out)
	}
	if strings.Contains(out, "WARNING") || strings.Contains(out, "EXPIRED") {
		t.Errorf("expected no warning or expiry at 2.1%%, got: %q", out)
	}
}

func TestConsumablesStatus_Warning(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	// 20.8 / 22 = 94.5% >= 90% threshold → WARNING
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 22}},
		},
	})

	out, err := runCmd(t, "consumables", "status")
	if err != nil {
		t.Fatalf("consumables status failed: %v", err)
	}

	if !strings.Contains(out, "WARNING") {
		t.Errorf("expected WARNING at 94.5%%, got: %q", out)
	}
}

func TestConsumablesStatus_Expired(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	// 20.8 / 20 = 104% → EXPIRED
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 20}},
		},
	})

	out, err := runCmd(t, "consumables", "status")
	if err != nil {
		t.Fatalf("consumables status failed: %v", err)
	}

	if !strings.Contains(out, "EXPIRED") {
		t.Errorf("expected EXPIRED at 104%%, got: %q", out)
	}
}

func TestConsumablesStatus_PerFilterWarningOverride(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	// 20.8 / 22 = 94.5%. Global warning is 90, but per-filter override is 99.
	// 94.5% < 99% → no warning should appear.
	w := 99
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 22, Warning: &w}},
		},
	})

	out, err := runCmd(t, "consumables", "status")
	if err != nil {
		t.Fatalf("consumables status failed: %v", err)
	}

	if strings.Contains(out, "WARNING") {
		t.Errorf("expected per-filter override to suppress WARNING at 94.5%% < 99%%, got: %q", out)
	}
}

func TestConsumablesStatus_DeviceIDFilter(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
			"dev2": {"sediment": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	out, err := runCmd(t, "consumables", "status", "--device-id", "dev1")
	if err != nil {
		t.Fatalf("consumables status --device-id dev1 failed: %v", err)
	}

	if !strings.Contains(out, "Device: dev1") {
		t.Errorf("expected dev1 in output, got: %q", out)
	}
	if strings.Contains(out, "Device: dev2") {
		t.Errorf("expected dev2 to be filtered out, got: %q", out)
	}
}

func TestConsumablesStatus_DeviceIDNotConfigured(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	_, err := runCmd(t, "consumables", "status", "--device-id", "unknown-dev")
	if err == nil {
		t.Fatal("expected error for unconfigured device ID, got nil")
	}
	if !strings.Contains(err.Error(), "unknown-dev") {
		t.Errorf("expected device ID in error message, got: %v", err)
	}
}

func TestConsumablesStatus_JSONOutput(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	// 20.8 / 1000 = 2.1% → ok
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	out, err := runCmd(t, "consumables", "status", "--output", "json")
	if err != nil {
		t.Fatalf("consumables status --output json failed: %v", err)
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, out)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	checks := map[string]any{
		"device_id":         "dev1",
		"name":              "charcoal",
		"status":            "ok",
		"warning_threshold": float64(90),
		"expires_gallons":   float64(1000),
	}
	for field, want := range checks {
		if r[field] != want {
			t.Errorf("%s = %v, want %v", field, r[field], want)
		}
	}
}

func TestConsumablesStatus_JSONShortFlag(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	// 20.8 / 22 = 94.5% → warning
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 22}},
		},
	})

	out, err := runCmd(t, "consumables", "status", "-o", "json")
	if err != nil {
		t.Fatalf("consumables status -o json failed: %v", err)
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, out)
	}
	if results[0]["status"] != "warning" {
		t.Errorf("status = %v, want warning", results[0]["status"])
	}
}

func TestConsumablesStatus_JSONExpiredStatus(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	// 20.8 / 20 = 104% → expired
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 20}},
		},
	})

	out, err := runCmd(t, "consumables", "status", "-o", "json")
	if err != nil {
		t.Fatalf("consumables status -o json failed: %v", err)
	}

	var results []map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &results); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %q", err, out)
	}
	if results[0]["status"] != "expired" {
		t.Errorf("status = %v, want expired", results[0]["status"])
	}
}

func TestConsumablesStatus_InvalidOutputFormat(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	_, err := runCmd(t, "consumables", "status", "--output", "csv")
	if err == nil {
		t.Fatal("expected error for unknown output format, got nil")
	}
	if !strings.Contains(err.Error(), "csv") {
		t.Errorf("expected format name in error, got: %v", err)
	}
}

func TestConsumablesStatus_PushGateway(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	var capturedMethod, capturedPath, capturedBody string
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		capturedBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(gateway.Close)

	out, err := runCmd(t, "consumables", "status", "--push-gateway", gateway.URL)
	if err != nil {
		t.Fatalf("consumables status --push-gateway failed: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected no stdout in push mode, got: %q", out)
	}
	if capturedMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", capturedMethod)
	}
	if capturedPath != "/metrics/job/flume_consumables" {
		t.Errorf("path = %q, want /metrics/job/flume_consumables", capturedPath)
	}
	for _, want := range []string{
		"flume_consumable_percent_used",
		"flume_consumable_used_gallons",
		"flume_consumable_expires_gallons",
		"flume_consumable_warning_threshold",
		`device_id="dev1"`,
		`name="charcoal"`,
	} {
		if !strings.Contains(capturedBody, want) {
			t.Errorf("body missing %q\nbody:\n%s", want, capturedBody)
		}
	}
}

func TestConsumablesStatus_PushGatewayGatewayError(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	t.Cleanup(gateway.Close)

	_, err := runCmd(t, "consumables", "status", "--push-gateway", gateway.URL)
	if err == nil {
		t.Fatal("expected error on gateway 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected status code in error, got: %v", err)
	}
}

func TestConsumablesStatus_PushGatewayOutputIncompatible(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
		Consumables: map[string]map[string]*config.Consumable{
			"dev1": {"charcoal": {Installed: mockInstalled, Expires: 1000}},
		},
	})

	_, err := runCmd(t, "consumables", "status", "--push-gateway", "http://localhost:9091", "--output", "json")
	if err == nil {
		t.Fatal("expected error when --push-gateway and --output are both set, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("expected 'mutually exclusive' in error, got: %v", err)
	}
}

func TestConsumablesStatus_NoConsumables(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)
	writeTestConfig(t, &config.Config{
		OutputFormat: "table",
		Warning:      90,
	})

	out, err := runCmd(t, "consumables", "status")
	if err != nil {
		t.Fatalf("consumables status failed: %v", err)
	}

	if !strings.Contains(out, "No consumables configured") {
		t.Errorf("expected help message, got: %q", out)
	}
}
