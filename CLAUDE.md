# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o bin/flume .          # build
go test -v -race ./...           # run all tests
go test -v -race ./cmd/...       # run a single package's tests
golangci-lint run                 # lint
gofmt -w .                       # format
goreleaser check                  # validate release config
goreleaser build --snapshot --clean  # local release build
```

## Architecture

`main.go` → `cmd/` (cobra commands) → `internal/` packages → `lib-flume-water` (external API client)

**`cmd/root.go`** is the structural hub. It owns:
- Package-level `cfg *config.Config` populated by `PersistentPreRunE` before every command runs.
- Package-level `newClient` func var — a factory for `*flumewater.Client`. Tests override this to inject a server pointed at an `httptest.Server`.
- `authenticate()` — checks the on-disk token cache first, falls through to a live API call on miss.
- `authenticateFresh()` — always does a live call (used by `auth` command to verify credentials).
- `resolveCredentials()` — credential priority: CLI flags → env vars → `.env` file.

**`internal/cache`** manages the JWT token cache at `$XDG_CONFIG_HOME/flume/token.json` (default `~/.config/flume/token.json`), written with `0600` permissions. `Load()` returns `nil` on any error (treated as cache miss). Cache failures on `Save()` are non-fatal — a warning is written to stderr and the command continues.

**`internal/config`** manages `~/.config/flume/config.yaml` (XDG-aware, `0600`). Fields: `output_format`, `default_device_id`, `default_location_id`. Missing file is auto-created with defaults; parse errors are fatal.

## Testing pattern

`cmd/testhelpers_test.go` provides the shared test infrastructure for all `cmd/` tests:
- `testServer(t, userID)` — starts an `httptest.Server` that handles all Flume API endpoints.
- `setupTestClient(t, serverURL)` — overrides `newClient`, sets credential env vars, redirects `XDG_CONFIG_HOME` to `t.TempDir()` (so tests never touch `~/.config/flume/token.json`), and registers cleanup to reset all package-level flag vars.
- `runCmd(t, args...)` — executes a cobra command and returns combined stdout+stderr output.

New command tests follow this pattern: call `setupTestClient(t, ts.URL)` with a test server, then call `runCmd(t, ...)` and assert on the returned string.
