# Release Process

This project uses [GoReleaser](https://goreleaser.com) to build cross-platform binaries and publish them as GitHub Releases. Releases are triggered automatically when a version tag is pushed.

## Versioning

This project follows [Semantic Versioning](https://semver.org/):

- **MAJOR** (`X.0.0`) — incompatible CLI or behavioral changes
- **MINOR** (`0.X.0`) — new commands or flags (backwards compatible)
- **PATCH** (`0.0.X`) — bug fixes

## Creating a Release

### 1. Ensure tests and lint pass

```bash
go test -v -race ./...
golangci-lint run
```

### 2. Commit all changes

```bash
git add .
git commit -m "chore: prepare release v0.2.0"
git push origin main
```

### 3. Create and push an annotated tag

```bash
git tag -a v0.2.0 -m "Release v0.2.0"
git push origin v0.2.0
```

Pushing the tag triggers the [release workflow](.github/workflows/release.yml), which:

1. Runs all tests
2. Builds binaries for Linux, macOS, and Windows (amd64 + arm64)
3. Creates a GitHub Release with changelog and checksums

### 4. Verify the release

1. Go to the **Actions** tab and confirm the release workflow succeeded.
2. Go to the **Releases** page and verify the artifacts and changelog look correct.

## Commit Message Conventions

Commits are automatically grouped in the release changelog:

| Prefix | Changelog section |
|--------|------------------|
| `feat:` | Features |
| `fix:` | Bug Fixes |
| `perf:` | Performance |
| `docs:`, `test:`, `chore:` | Excluded |

## Local Release Testing

Test the release build without publishing:

```bash
goreleaser release --snapshot --clean
```

Artifacts land in `./dist/`.

## Required Repository Settings

- **Workflow permissions**: Settings → Actions → General → "Read and write permissions"
- No additional secrets are required; the release workflow uses the built-in `GITHUB_TOKEN`.
