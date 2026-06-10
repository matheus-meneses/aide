# cmd/aide

## Purpose

CLI entrypoint and command definitions using Cobra. Each file defines one or more subcommands that orchestrate internal packages. Commands are thin wrappers — business logic lives in internal packages.

## Structure

| File | Commands |
|------|----------|
| `main.go` | Root command, global flags (`--config`), version check hook |
| `initcmd.go` | `aide init` — setup ~/.aide, extract scrapers, create venv, download registry |
| `run.go` | `aide run` — execute scrapers via runner package |
| `agent.go` | `aide agent start` / `aide agent status` — start autonomous agent |
| `configcmd.go` | `aide config show` / `aide config check` — display and validate config |
| `sourcecmd.go` | `aide config source add` / `aide config source list` — interactive source setup |
| `credential.go` | `aide credential set` / `aide credential list` / `aide credential delete` |
| `report.go` | `aide report` — print open items |
| `diff.go` | `aide diff` — show 24h changes |
| `stats.go` | `aide stats` — historical stats with sparklines |
| `sources.go` | `aide sources` — source health overview |
| `history.go` | `aide history` — run history table |
| `prune.go` | `aide prune` — data retention cleanup |
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
- `plugin install`/`plugin update` resolve the registry index from a GitHub release. The source repo is `AIDE_REGISTRY_REPO` (default `matheus-meneses/aide-plugins`); the release is `latest` unless pinned with `AIDE_REGISTRY_VERSION` or `--registry-version <tag>` (e.g. `v0.1.0-rc1`, which `latest` would skip as a prerelease). The version-pinned index and its per-plugin tarballs share the same tag, so SHA-256 verification stays consistent.
- Private registries: when a token is present (`GH_TOKEN`/`GITHUB_TOKEN`/`gh auth token`), index and artifact downloads go through the GitHub release-asset API instead of the `releases/download` browser URLs, which require a session for private repos.
- `source add` surfaces a user-friendly message when all sources are configured (not a usage error).

## Relations

- Depends on: all internal packages (`config`, `store`, `runner`, `render`, `agent`, `keychain`, `registry`, `prompt`, `scrapers`, `updater`)
