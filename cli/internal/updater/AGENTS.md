# updater

## Purpose

Version checking against internal Nexus repository and file download utilities. Prints a stderr banner when a newer version is available.

## Exported API

- `NexusBaseURL` — constant: `https://nexus.sharedservices.local/repository/aide`
- `CheckOnce(currentVersion string)` — throttled version check (12h cooldown)
- `DownloadFile(url string, dest *os.File, showProgress bool) error`
- `DownloadToPath(url, destPath string) error`

## Important Invariants

- `CheckOnce` reads/writes `~/.aide/.last_version_check` to throttle to max once per 12 hours.
- Skips entirely when `currentVersion == "dev"`.
- HTTP client uses 3s timeout and `InsecureSkipVerify: true` (internal corporate CA).
- Version comparison is simple string inequality (`!=`), not semver.
- Banner prints to stderr so it doesn't interfere with CLI pipe output.

## Pitfalls

- TLS skip verify is intentional for corporate proxy — do not remove without verifying CA trust chain.
- `DownloadFile` overwrites the destination without checking existing content.
- Network failures are silently swallowed in `CheckOnce` (by design — non-blocking).
- `DownloadToPath` creates parent directories but uses 0o644 permissions.

## Relations

- Used by: `cmd/aide` (main entrypoint calls `CheckOnce`), `initcmd` (downloads registry), `agent/server` (version endpoint)
- No internal dependencies (leaf package).
