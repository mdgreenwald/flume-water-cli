// Package cache manages the on-disk token cache for the Flume CLI.
// Tokens are stored at $XDG_CONFIG_HOME/flume/token.json (defaulting to
// $HOME/.config/flume/token.json) with permissions 0600.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	flumewater "github.com/mdgreenwald/lib-flume-water"
)

// expiryBuffer is how far before the actual expiry we consider a token stale.
// This guards against clock skew and latency between the cache check and the
// first API call that uses the token.
const expiryBuffer = 5 * time.Minute

// TokenCache is the cached authentication result persisted to disk.
type TokenCache struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	UserID       string    `json:"user_id"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IsValid reports whether the cached token has at least expiryBuffer of
// lifetime remaining.
func (t *TokenCache) IsValid() bool {
	return time.Until(t.ExpiresAt) > expiryBuffer
}

// ToAuthResult converts the cached entry into the type expected by the library.
func (t *TokenCache) ToAuthResult() *flumewater.AuthResult {
	return &flumewater.AuthResult{
		AccessToken:  t.AccessToken,
		RefreshToken: t.RefreshToken,
		UserID:       t.UserID,
	}
}

// Dir returns the flume configuration directory following the XDG Base
// Directory specification. If $XDG_CONFIG_HOME is set it is used; otherwise
// the path defaults to $HOME/.config/flume.
func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "flume"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".config", "flume"), nil
}

// tokenPath returns the full path to the token cache file.
func tokenPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// Load reads the token cache from disk. It returns nil without an error when
// the cache file does not exist or cannot be parsed — callers should treat nil
// as a cache miss and fall through to a live authentication.
func Load() *TokenCache {
	path, err := tokenPath()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var tc TokenCache
	if err := json.Unmarshal(data, &tc); err != nil {
		return nil
	}
	return &tc
}

// Save writes the authentication result to the token cache file, creating the
// directory if necessary. The expiry comes from AuthResult.ExpiresAt (set by
// the lib from the JWT exp claim). A zero ExpiresAt falls back to one hour.
func Save(result *flumewater.AuthResult) error {
	expiresAt := result.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(time.Hour)
	}

	tc := TokenCache{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		UserID:       result.UserID,
		ExpiresAt:    expiresAt,
	}

	data, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal token cache: %w", err)
	}

	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, "token.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write token cache %s: %w", path, err)
	}
	return nil
}
