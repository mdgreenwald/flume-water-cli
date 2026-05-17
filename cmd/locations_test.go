package cmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	flumewater "github.com/mdgreenwald/lib-flume-water"
)

func TestLocationsListCommand(t *testing.T) {
	ts := testServer(t, testUserID)
	setupTestClient(t, ts.URL)

	out, err := runCmd(t, "locations", "list")
	if err != nil {
		t.Fatalf("locations list command failed: %v", err)
	}

	if !strings.Contains(out, "Home") {
		t.Errorf("expected location name 'Home' in output, got: %q", out)
	}
	if !strings.Contains(out, "100") {
		t.Errorf("expected location ID '100' in output, got: %q", out)
	}
	if !strings.Contains(out, "Springfield") {
		t.Errorf("expected city 'Springfield' in output, got: %q", out)
	}
	if !strings.Contains(out, "America/Los_Angeles") {
		t.Errorf("expected timezone in output, got: %q", out)
	}
}

func TestLocationsListCommand_Empty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/oauth/token"):
			writeJSON(w, map[string]any{
				"success": true, "code": 200, "message": "OK",
				"data": []map[string]any{{
					"token_type": "bearer", "access_token": testJWT(testUserID),
					"expires_in": 3600, "refresh_token": "r", "user_id": testUserID,
				}},
			})
		default:
			writeJSON(w, map[string]any{
				"success": true, "code": 200, "message": "OK", "count": 0, "data": []any{},
			})
		}
	}))
	t.Cleanup(ts.Close)

	orig := newClient
	newClient = func() *flumewater.Client {
		c := flumewater.NewClient()
		c.BaseURL = ts.URL
		return c
	}
	t.Cleanup(func() { newClient = orig })

	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("FLUME_CLIENT_ID", "test-client-id")
	t.Setenv("FLUME_CLIENT_SECRET", "test-client-secret")
	t.Setenv("FLUME_USER_EMAIL", "test@example.com")
	t.Setenv("FLUME_USER_PASSWORD", "test-password")

	out, err := runCmd(t, "locations", "list")
	if err != nil {
		t.Fatalf("locations list command failed: %v", err)
	}
	if !strings.Contains(out, "No locations found") {
		t.Errorf("expected 'No locations found', got: %q", out)
	}
}
