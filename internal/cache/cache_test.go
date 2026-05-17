package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	flumewater "github.com/mdgreenwald/lib-flume-water"
)

func TestDir_XDGOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	want := filepath.Join(tmp, "flume")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestDir_DefaultsToHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir() error: %v", err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "flume")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestIsValid(t *testing.T) {
	cases := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"valid — far future", time.Now().Add(time.Hour), true},
		{"valid — just past buffer", time.Now().Add(expiryBuffer + time.Second), true},
		{"invalid — within buffer", time.Now().Add(expiryBuffer - time.Second), false},
		{"invalid — expired", time.Now().Add(-time.Hour), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tok := &TokenCache{ExpiresAt: tc.expiresAt}
			if got := tok.IsValid(); got != tc.want {
				t.Errorf("IsValid() = %v, want %v (expires in %v)", got, tc.want, time.Until(tc.expiresAt))
			}
		})
	}
}

func TestToAuthResult(t *testing.T) {
	tc := &TokenCache{
		AccessToken:  "at",
		RefreshToken: "rt",
		UserID:       "42",
	}
	got := tc.ToAuthResult()
	if got.AccessToken != tc.AccessToken || got.RefreshToken != tc.RefreshToken || got.UserID != tc.UserID {
		t.Errorf("ToAuthResult() = %+v, does not match source %+v", got, tc)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	future := time.Now().Add(time.Hour).UTC().Truncate(time.Second)
	result := &flumewater.AuthResult{
		AccessToken:  "test-access-token",
		RefreshToken: "refresh-token",
		UserID:       "99",
		ExpiresAt:    future,
	}

	if err := Save(result); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded := Load()
	if loaded == nil {
		t.Fatal("Load() returned nil after Save()")
	}
	if loaded.AccessToken != result.AccessToken {
		t.Errorf("AccessToken mismatch: got %q, want %q", loaded.AccessToken, result.AccessToken)
	}
	if loaded.RefreshToken != result.RefreshToken {
		t.Errorf("RefreshToken mismatch: got %q, want %q", loaded.RefreshToken, result.RefreshToken)
	}
	if loaded.UserID != result.UserID {
		t.Errorf("UserID mismatch: got %q, want %q", loaded.UserID, result.UserID)
	}
	if loaded.ExpiresAt.Unix() != future.Unix() {
		t.Errorf("ExpiresAt mismatch: got %v, want %v", loaded.ExpiresAt, future)
	}
}

func TestSave_CreatesDirectoryAndSetsPermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	result := &flumewater.AuthResult{
		AccessToken: "test-access-token",
		UserID:      "1",
		ExpiresAt:   time.Now().Add(time.Hour),
	}
	if err := Save(result); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	path := filepath.Join(tmp, "flume", "token.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("token.json not found: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("token.json permissions = %o, want 0600", perm)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if tc := Load(); tc != nil {
		t.Errorf("Load() = %+v, want nil for missing cache", tc)
	}
}

func TestLoad_CorruptFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	dir := filepath.Join(tmp, "flume")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "token.json"), []byte("not json"), 0o600)

	if tc := Load(); tc != nil {
		t.Errorf("Load() = %+v, want nil for corrupt cache", tc)
	}
}

func TestSave_FallbackExpiryWhenExpiresAtZero(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	before := time.Now()
	result := &flumewater.AuthResult{
		AccessToken: "test-access-token",
		UserID:      "1",
		// ExpiresAt intentionally zero — triggers the 1-hour fallback.
	}
	if err := Save(result); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded := Load()
	if loaded == nil {
		t.Fatal("Load() returned nil")
	}
	if loaded.ExpiresAt.Before(before.Add(55 * time.Minute)) {
		t.Errorf("fallback ExpiresAt %v is too early", loaded.ExpiresAt)
	}
}

// TestSave_ContentIsValidJSON confirms the written file is readable JSON.
func TestSave_ContentIsValidJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	if err := Save(&flumewater.AuthResult{AccessToken: "test-access-token", UserID: "1", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	dir, _ := Dir()
	data, err := os.ReadFile(filepath.Join(dir, "token.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("token.json is not valid JSON: %v", err)
	}
}
