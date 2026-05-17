# flume-water-cli

A Go CLI tool for interacting with the [Flume Water API](https://flumewater.com) via [lib-flume-water](https://github.com/mdgreenwald/lib-flume-water).

## Installation

### Download a pre-built binary

Download the latest release from [GitHub Releases](https://github.com/mdgreenwald/flume-water-cli/releases) for your platform (Linux, macOS, Windows — amd64 and arm64).

### Build from source

```bash
go install github.com/mdgreenwald/flume-water-cli@latest
```

## Configuration

### Credentials

The CLI needs your Flume API credentials. Provide them in any of the following ways (highest priority first):

1. **Command-line flags** — `--client-id`, `--client-secret`, `--email`, `--password`
2. **Environment variables** — `FLUME_CLIENT_ID`, `FLUME_CLIENT_SECRET`, `FLUME_USER_EMAIL`, `FLUME_USER_PASSWORD`
3. **`.env` file** — pass `--env-file /path/to/.env` or place a `.env` file in the working directory

Example `.env` file:

```env
FLUME_CLIENT_ID=your_client_id
FLUME_CLIENT_SECRET=your_client_secret
FLUME_USER_EMAIL=you@example.com
FLUME_USER_PASSWORD=your_password
```

You can get API credentials from the [Flume Water portal](https://portal.flumewater.com).

### Token caching

After a successful authentication the access token is cached at:

```
$XDG_CONFIG_HOME/flume/token.json   # if XDG_CONFIG_HOME is set
~/.config/flume/token.json          # default
```

The file is created with permissions `0600`. Subsequent commands reuse the cached token until it expires (the CLI reads the `exp` claim from the JWT), so most invocations avoid a round-trip to the auth endpoint.

**Cron job setup** — run `flume auth` once interactively with your credentials to prime the cache, then schedule the usage check without needing credentials in the cron environment:

```cron
0 7 * * * /usr/local/bin/flume usage query --device-id <id> >> /var/log/flume.log
```

When the token expires the cron command re-authenticates automatically if credentials are reachable (env vars or `.env`); otherwise it exits with a descriptive error.

To reset the cache (e.g. after a credential rotation), delete the file:

```bash
rm ~/.config/flume/token.json
```

## Usage

### Test authentication

```bash
flume auth
```

### List locations

```bash
flume locations list
```

### List devices

```bash
# All devices
flume devices list

# Devices at a specific location
flume devices list --location-id <location-id>
```

### Query water usage

```bash
# Last 30 days of daily usage for a device
flume usage query --device-id <device-id>

# Last 7 days of hourly usage
flume usage query --device-id <device-id> --days 7 --bucket HOUR
```

Available `--bucket` values: `MIN`, `HOUR`, `DAY`, `WEEK`, `MON`, `YR`

### Version

```bash
flume --version
```

## Development

### Requirements

- [Go 1.26+](https://go.dev/dl/)
- [mise](https://mise.jdx.dev/) (optional, installs all tools)

```bash
mise install
```

### Run tests

```bash
go test -v -race ./...
```

### Lint

```bash
golangci-lint run
```

### Format

```bash
gofmt -w .
```

### Build

```bash
go build -o bin/flume .
```

### Validate release config

```bash
goreleaser check
goreleaser build --snapshot --clean
```

## Releasing

See [RELEASE.md](RELEASE.md) for the release process.

## License

[MIT](LICENSE)
