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

### Track consumable life spans

Add consumables to `~/.config/flume/config.yaml` (use `flume devices list` to find your device IDs):

```yaml
warning: 90   # warn at 90% consumed (default); override per-filter below

consumables:
  "111111111111111111":
    charcoal:
      installed: 2026-03-15
      expires: 15000        # gallons
    sediment:
      installed: 2026-03-15
      expires: 10000
      warning: 85           # override: warn earlier for this filter
```

```bash
# Show remaining life for all configured filters
flume consumables status

# Show a specific device only
flume consumables status --device-id <device-id>

# Output as JSON (pipe to jq)
flume consumables status --output json
flume consumables status --output json | jq '.[] | select(.status == "warning")'
flume consumables status --output json | jq '.[] | select(.device_id == "<id>" and .name == "charcoal")'

# Push metrics to a Prometheus Pushgateway
flume consumables status --push-gateway http://localhost:9091
```

Output shows gallons used, total capacity, percentage, and a warning or expiry marker:

```
Device: 111111111111111111
  charcoal       installed: 2026-03-15    3,247.0 / 15000 gal (21.6%)
  sediment       installed: 2026-03-15    9,150.0 / 10000 gal (91.5%)  WARNING: approaching end of life
```

JSON output fields: `device_id`, `name`, `installed`, `expires_gallons`, `used_gallons`, `percent_used`, `warning_threshold`, `status` (`ok`, `warning`, or `expired`).

#### Prometheus / Grafana integration

Push metrics on a schedule using cron — the Pushgateway retains the last value between pushes:

```cron
*/5 * * * * /usr/local/bin/flume consumables status --push-gateway http://localhost:9091
```

Exposed metrics (all gauges, labeled by `device_id` and `name`):

| Metric | Description |
|---|---|
| `flume_consumable_percent_used` | Percentage of consumable life used |
| `flume_consumable_used_gallons` | Gallons consumed since installation |
| `flume_consumable_expires_gallons` | Total gallon capacity |
| `flume_consumable_warning_threshold` | Warning threshold percentage |

`--push-gateway` and `--output` are mutually exclusive.

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
