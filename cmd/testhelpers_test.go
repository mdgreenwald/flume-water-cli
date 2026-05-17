package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	flumewater "github.com/mdgreenwald/lib-flume-water"
)

// testUserID is the user ID used across test fixtures.
const testUserID = "99999"

// testJWT returns a JWT with the given userID embedded in the payload.
// Signature verification is disabled in lib-flume-water (jwt.WithVerify(false)),
// so only a structurally valid token is required.
func testJWT(userID string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		fmt.Sprintf(`{"user_id":"%s","exp":9999999999}`, userID),
	))
	return header + "." + payload + ".test-signature"
}

// mustMarshal marshals v and panics on failure — only for constructing test fixtures.
func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// writeJSON writes JSON to a ResponseWriter and ignores the return values
// since errors writing to an in-process test server are unactionable.
func writeJSON(w http.ResponseWriter, v any) {
	_, _ = fmt.Fprint(w, mustMarshal(v))
}

// testServer starts an httptest server that handles all Flume API endpoints used by the CLI.
func testServer(t *testing.T, userID string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/oauth/token"):
			writeJSON(w, map[string]any{
				"success": true,
				"code":    200,
				"message": "OK",
				"data": []map[string]any{{
					"token_type":    "bearer",
					"access_token":  testJWT(userID),
					"expires_in":    3600,
					"refresh_token": "test-refresh-token",
					"user_id":       userID,
				}},
			})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/locations"):
			writeJSON(w, map[string]any{
				"success": true,
				"code":    200,
				"message": "OK",
				"count":   1,
				"data": []map[string]any{{
					"id":          100,
					"name":        "Home",
					"address":     "123 Main St",
					"city":        "Springfield",
					"state":       "CA",
					"postal_code": "90210",
					"tz":          "America/Los_Angeles",
					"user_id":     99999,
				}},
			})

		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/devices"):
			writeJSON(w, map[string]any{
				"success": true,
				"code":    200,
				"message": "OK",
				"count":   2,
				"data": []map[string]any{
					{
						"id":        "111111111111111111",
						"type":      1,
						"connected": true,
						"last_seen": "2026-01-15T10:30:00Z",
						"location":  map[string]any{"id": 100},
					},
					{
						"id":        "222222222222222222",
						"type":      2,
						"connected": true,
						"last_seen": "2026-01-15T10:31:00Z",
						"location":  map[string]any{"id": 100},
					},
				},
			})

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/query"):
			writeJSON(w, map[string]any{
				"success": true,
				"code":    200,
				"message": "OK",
				"data": []map[string]any{{
					"usage": []map[string]any{
						{"datetime": "2026-01-01 00:00:00", "value": 12.5},
						{"datetime": "2026-01-02 00:00:00", "value": 8.3},
					},
				}},
			})

		default:
			http.Error(w, `{"success":false,"message":"not found"}`, http.StatusNotFound)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

// setupTestClient overrides the newClient factory to point at the given test server URL
// and sets credential env vars so authenticate() resolves without a .env file.
func setupTestClient(t *testing.T, serverURL string) {
	t.Helper()

	orig := newClient
	newClient = func() *flumewater.Client {
		c := flumewater.NewClient()
		c.BaseURL = serverURL
		return c
	}

	// Redirect the token cache to a temp dir so tests never read or write
	// the real ~/.config/flume/token.json.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	t.Setenv("FLUME_CLIENT_ID", "test-client-id")
	t.Setenv("FLUME_CLIENT_SECRET", "test-client-secret")
	t.Setenv("FLUME_USER_EMAIL", "test@example.com")
	t.Setenv("FLUME_USER_PASSWORD", "test-password")

	t.Cleanup(func() {
		newClient = orig
		envFileFlag = ""
		clientIDFlag = ""
		clientSecretFlag = ""
		emailFlag = ""
		passwordFlag = ""
		locationIDFlag = ""
		deviceIDFlag = ""
		daysFlag = 30
		bucketFlag = "DAY"
	})
}

// runCmd executes the given cobra command args and returns the combined output.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}
