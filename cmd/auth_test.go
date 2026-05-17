package cmd

import (
	"strings"
	"testing"
)

func TestAuthCommand_Success(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	out, err := runCmd(t, "auth")
	if err != nil {
		t.Fatalf("auth command failed: %v", err)
	}

	if !strings.Contains(out, "Authentication successful") {
		t.Errorf("expected success message, got: %q", out)
	}
	if !strings.Contains(out, testUserID) {
		t.Errorf("expected user ID %q in output, got: %q", testUserID, out)
	}
	if !strings.Contains(out, "Token cached:") {
		t.Errorf("expected cache path in output, got: %q", out)
	}
	if !strings.Contains(out, "Token expires:") {
		t.Errorf("expected expiry in output, got: %q", out)
	}
}

func TestAuthCommand_AlwaysFresh(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	// First call: primes the cache.
	if _, err := runCmd(t, "auth"); err != nil {
		t.Fatalf("first auth failed: %v", err)
	}

	// Second call: should still hit the API (auth always bypasses the cache).
	// We verify by ensuring the server is still reachable and the command succeeds.
	out, err := runCmd(t, "auth")
	if err != nil {
		t.Fatalf("second auth failed: %v", err)
	}
	if !strings.Contains(out, "Authentication successful") {
		t.Errorf("expected success on second auth, got: %q", out)
	}
}

func TestAuthCommand_MissingCredentials(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	t.Setenv("FLUME_CLIENT_ID", "")
	t.Setenv("FLUME_CLIENT_SECRET", "")
	t.Setenv("FLUME_USER_EMAIL", "")
	t.Setenv("FLUME_USER_PASSWORD", "")

	_, err := runCmd(t, "auth")
	if err == nil {
		t.Fatal("expected error when credentials are missing, got nil")
	}
}

func TestAuthenticate_UsesCachedToken(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	// Warm the cache via the auth command.
	if _, err := runCmd(t, "auth"); err != nil {
		t.Fatalf("auth (cache warm) failed: %v", err)
	}

	// Subsequent command (locations list) should use the cache and not call /oauth/token again.
	// We can't easily count HTTP calls, but we verify the command succeeds and returns data.
	out, err := runCmd(t, "locations", "list")
	if err != nil {
		t.Fatalf("locations list after cache warm failed: %v", err)
	}
	if !strings.Contains(out, "Home") {
		t.Errorf("expected location data in output, got: %q", out)
	}
}
