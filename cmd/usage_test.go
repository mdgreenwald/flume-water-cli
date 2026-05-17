package cmd

import (
	"strings"
	"testing"
)

func TestUsageQueryCommand(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	out, err := runCmd(t, "usage", "query", "--device-id", "dev-sensor-1")
	if err != nil {
		t.Fatalf("usage query command failed: %v", err)
	}

	if !strings.Contains(out, "gallons") {
		t.Errorf("expected 'gallons' in output, got: %q", out)
	}
	if !strings.Contains(out, "2026-01-01") {
		t.Errorf("expected date '2026-01-01' in output, got: %q", out)
	}
}

func TestUsageQueryCommand_MissingDeviceID(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	_, err := runCmd(t, "usage", "query")
	if err == nil {
		t.Fatal("expected error when --device-id is missing, got nil")
	}
	if !strings.Contains(err.Error(), "--device-id") {
		t.Errorf("expected error to mention --device-id, got: %v", err)
	}
}

func TestUsageQueryCommand_CustomFlags(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	out, err := runCmd(t, "usage", "query", "--device-id", "dev-sensor-1", "--days", "7", "--bucket", "HOUR")
	if err != nil {
		t.Fatalf("usage query with custom flags failed: %v", err)
	}

	if !strings.Contains(out, "gallons") {
		t.Errorf("expected usage data in output, got: %q", out)
	}
}
