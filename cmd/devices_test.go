package cmd

import (
	"strings"
	"testing"
)

func TestDevicesListCommand(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	out, err := runCmd(t, "devices", "list")
	if err != nil {
		t.Fatalf("devices list command failed: %v", err)
	}

	if !strings.Contains(out, "Bridge") {
		t.Errorf("expected device type 'Bridge' in output, got: %q", out)
	}
	if !strings.Contains(out, "Sensor") {
		t.Errorf("expected device type 'Sensor' in output, got: %q", out)
	}
	if !strings.Contains(out, "111111111111111111") {
		t.Errorf("expected bridge device ID in output, got: %q", out)
	}
	if !strings.Contains(out, "222222222222222222") {
		t.Errorf("expected sensor device ID in output, got: %q", out)
	}
}

func TestDevicesListCommand_FilterByLocation(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	out, err := runCmd(t, "devices", "list", "--location-id", "100")
	if err != nil {
		t.Fatalf("devices list --location-id command failed: %v", err)
	}

	if !strings.Contains(out, "Bridge") && !strings.Contains(out, "Sensor") {
		t.Errorf("expected at least one device in output, got: %q", out)
	}
}
