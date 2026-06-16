# updater

## Purpose

Version checking against the GitHub releases endpoint and file download utilities. Prints a stderr banner when a newer version is available.

## Exported API

- `CheckOnce(currentVersion string)` — throttled version check (12h cooldown)
- `DownloadFile(url string, dest *os.File, showProgress bool) error`
- `DownloadToPath(url, destPath string) error`

## Important Invariants

- `CheckOnce` reads/writes `~/.aide/.last_version_check` to throttle to max once per 12 hours.
- Skips entirely when `currentVersion == "dev"`.
- Release base URL defaults to `https://github.com/matheus-meneses/aide/releases/latest/download`; override with `AIDE_RELEASE_URL` env var.
- HTTP client uses 5s timeout with standard TLS.
- Version comparison is simple string inequality (`!=`), not semver.
- Banner prints to stderr so it does not interfere with CLI pipe output.

## Pitfalls

- Network failures are silently swallowed in `CheckOnce` (by design — non-blocking).
- `DownloadFile` overwrites the destination without checking existing content.
- `DownloadToPath` creates parent directories but uses 0o644 permissions.

## Relations

- Used by: `cmd/aide` (main entrypoint calls `CheckOnce`), `initcmd` (downloads registry), `agent/server` (version endpoint)
- No internal dependencies (leaf package).
