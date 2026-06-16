# cmd/aide

## Purpose

CLI entrypoint and command definitions using Cobra. Each file defines one or more subcommands that orchestrate internal packages. Commands are thin wrappers — business logic lives in internal packages.

## Structure

| File | Commands |
|------|----------|
| `main.go` | Root command, global flags (`--config`, `--verify-ssl`, `--ca-bundle`), version check hook |
| `initcmd.go` | `aide init` — setup ~/.aide, extract scrapers, create venv, download registry |
| `uicmd.go` | `aide ui` — serve the web UI + run the autonomous agent (port 8531, `--no-browser`) |
| `run.go` | `aide run` — execute scrapers via runner package |
| `agent.go` | `aide agent start` (headless foreground loop) / `aide agent status` / `aide agent ask` / `aide agent schedule` |
| `configcmd.go` | `aide config show` / `aide config check` / `aide config set` — display and validate config |
| `plugincmd.go` | `aide plugin list/install/configure/enable/disable/set/remove/status` + `plugin registry` subtree |
| `plugincmd_configure.go` | `aide plugin configure` — interactive source setup (settings + credentials) |
| `plugincmd_registry.go` | `aide plugin registry list/add/remove/refresh` — manage plugin registries |
| `plugincmd_list.go` | `aide plugin list` — installed plugins with status (`--available` for the catalog) |
| `credential.go` | `aide credential set` / `aide credential list` / `aide credential delete` |
| `report.go` | `aide report` — print open items |
| `diff.go` | `aide diff` — show 24h changes |
| `stats.go` | `aide stats` — historical stats with sparklines |
| `history.go` | `aide history` — run history table |
| `prune.go` | `aide prune` — data retention cleanup |
| `tlscmd.go` | `aide tls fetch <host>` — TOFU capture of a server chain into `~/.aide/certs/`, optional config wiring |
| `versioncmd.go` | `aide version` — print version |
| `whoami.go` | `aide whoami` — show resolved identity |

## Important Invariants

- Global `cfgFile` flag defaults to `~/.aide/config.yaml`.
- `PersistentPostRun` calls `updater.CheckOnce(version)` — throttled to once per 12h.
- Commands that need store always `defer s.Close()`.
- `config check` returns non-zero exit on issues.
- `run` returns non-zero if any source fails.
- `run --source` validates names before execution via `runner.ValidateFilter`.
- Default config template values match `config.Load()` defaults (concurrency: 5, run_interval: "30m").

## Pitfalls

- Many commands repeat `config.Load` + `store.Open` + `defer Close` boilerplate — could be extracted to a helper in the future.
- `version` var is set via `-ldflags` at build time; defaults to `"dev"`.
- `initcmd` uses network access to download the registry; defaults to GitHub releases. Override with `AIDE_RELEASE_URL` env var.
- `plugin install`/`plugin registry refresh` resolve the registry index from a GitHub release. The source repo is `AIDE_REGISTRY_REPO` (default `matheus-meneses/aide-plugins`); the release is `latest` unless pinned with `AIDE_REGISTRY_VERSION` or `--registry-version <tag>` (e.g. `v0.1.0-rc1`, which `latest` would skip as a prerelease). The version-pinned index and its per-plugin tarballs share the same tag, so SHA-256 verification stays consistent.
- Private registries: when a token is present (`GH_TOKEN`/`GITHUB_TOKEN`/`gh auth token`), index and artifact downloads go through the GitHub release-asset API instead of the `releases/download` browser URLs, which require a session for private repos.
- `plugin configure` surfaces a user-friendly message when all sources are configured (not a usage error).
- TLS: `--verify-ssl`/`--ca-bundle` only override config when the flag was `Changed` (so `run` honors `config.yaml` per-source/global `tls:` for unattended agent runs). The runner resolves flag > per-source > global > secure default and injects `verify_ssl` + `ca_bundle`. `tls fetch` dials with `InsecureSkipVerify` on purpose — it's trust-on-first-use, so it prints SHA-256 fingerprints to verify out-of-band.
- macOS sandbox: plugins run under `sandbox-exec` with no Mach access, so `truststore` can't reach `trustd` (symptom: `unable to get local issuer certificate`). The runner works around this by exporting the OS trust store to `~/.aide/cache/system-trust.pem` (`trust.SystemBundle()`, in `internal/runtime/trust`) and injecting it as `ca_bundle` when verification is on and no explicit bundle is set; OpenSSL then verifies via file-read inside the sandbox. The same bundle is fed to Node (`NODE_EXTRA_CA_CERTS`) when downloading Playwright browsers during plugin install, which otherwise hits the same `unable to get local issuer certificate` behind a TLS-intercepting proxy. Note `--verify-ssl false` (space) does NOT bypass — cobra needs `--verify-ssl=false`.

## Relations

- Depends on: all internal packages (`config`, `store`, `runner`, `render`, `agent`, `keychain`, `registry`, `prompt`, `scrapers`, `updater`)
