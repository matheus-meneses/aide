# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Plugin updates** — `aide plugin update [name]` upgrades one or every
  installed plugin to the latest version published in the configured
  registries, rebuilding its runtime in place while preserving config and
  stored credentials (`--check` to only report what would change). `aide plugin
  list` now flags plugins with an update available. In the web UI, the
  Marketplace and Installed tabs show an `update → vX.Y.Z` badge and a one-click
  **Update** button.
- **Plugin source & icon in the catalog** — the Marketplace tags each plugin as
  `builtin` (default catalog) or `private` (a user-added registry), and the
  plugin manifest gains an optional `icon` field (an emoji, a URL, or a data
  URI) rendered in the catalog and installed lists.

### Fixed

- Installing or updating a plugin now wipes the previous install directory
  before extracting, so files removed in a newer version no longer linger.

## [0.2.0] - 2026-06-17

0.2.0 brings a **native macOS desktop app**, a streamlined CLI — a new `aide ui`
launcher and a consolidated `aide plugin` command tree — built-in **auto-update**,
and a round of plugin/sandbox security hardening.

### Added

- **Auto-update** for both the CLI and the macOS app. aide now detects how it
  was installed (install script, Homebrew formula, Homebrew cask, or a manually
  installed `.app`) and updates itself accordingly: standalone installs
  self-replace the binary or swap the app bundle in place and relaunch, while
  Homebrew installs run `brew upgrade` transparently so brew's state stays
  consistent. Updates are sha256-verified before being applied.
- **`aide update`** command — checks for a newer release, shows the release
  notes, and applies the update in place (`--check` to only report).
- **One-click "Update now"** in the web UI banner, plus a new **About** settings
  tab showing the current version, platform, the changelog, and an
  `auto_update` preference (`off` / `notify` / `auto`).
- **Native desktop app (macOS)** — a single Aide.app that bundles the agent and
  the web UI, with a guided first-run setup and a redesigned interface. Install
  with Homebrew (`brew install --cask aide`) or the `.dmg`.
- **Live logs in the app** — tail the agent's log file in real time over SSE,
  with on-demand log pruning, so you can see what every source and plugin is
  doing without leaving the UI.
- **Go unit-test suite** covering config, plugin loading/validation, the scrape
  runner, and the per-OS sandbox policy builders. CI now runs tests with the
  race detector and coverage.
- **`aide ui`** — a desktop-equivalent launcher that serves the web UI and runs
  the autonomous agent in one command (defaults to port 8531, opens the browser,
  `--no-browser` to skip). Works without a prior `aide init`: the in-browser
  setup wizard handles first-run configuration.

### Changed

- **Unified logging** — all runner and plugin output now flows through one
  structured logger (`clog`) with consistent levels and `text`/`json` formats
  across the CLI and the desktop app.
- **CLI reorganised around `aide plugin`** — `aide sources`, `aide registry`, and
  the `aide config source` subtree were folded into a single `aide plugin` tree
  (`list`, `install`, `configure`, `enable`, `disable`, `set`, `remove`,
  `status`, `registry`). `aide plugin update` and `aide plugin auth` were
  removed (use `aide plugin registry refresh`; browser login is now part of the
  scrape flow). This is a clean break with no deprecated aliases.
- **`aide agent start` is now a headless foreground runner** — it runs the
  autonomous loop with no HTTP server or browser. Use `aide ui` for the web
  experience.
- **Web UI rebranded** to lowercase `aide` via a single `APP_NAME` constant.
- **Codebase reorganised into concept domains** with enforced import boundaries
  (depguard rules plus an architecture test), keeping the agent, runtime, UI,
  and platform layers cleanly separated.

### Fixed

- Web UI no longer drops the active conversation when switching to Settings or
  an items view.
- Cleared all frontend lint warnings and tightened TypeScript strictness.
- Browser-based plugins no longer crash with `SIGSEGV` on macOS: the deny-default
  sandbox profile could not host a full browser engine, so browser plugins now
  use a relaxed, write-confined profile.
- Browser plugins no longer fail on fresh installs with a missing Playwright
  `chrome-headless-shell`: the installer now always runs Playwright's idempotent
  browser install for plugins that need it, instead of skipping when the cache
  directory merely exists.
- The web UI can now configure `object_list` plugin fields (such as Jira's JQL
  queries) with an inline add/remove editor, instead of a dead-end note pointing
  at a removed command. Saving such a source no longer fails with a spurious
  "missing required config field" error.
- Playwright browser downloads during plugin install now trust the configured
  `ca_bundle` (or the exported OS trust store) via `NODE_EXTRA_CA_CERTS`, fixing
  `unable to get local issuer certificate` failures behind TLS-intercepting
  corporate proxies. `verify_ssl: false` disables verification for the download.
- `aide ui` no longer spams `unsupported protocol scheme ""` LLM errors when
  started before a model is configured. The autonomous loop now stays idle until
  setup is complete and kicks off automatically once a model is saved.
- The web chat no longer shows an empty "Nothing to show." card while a reply is
  still streaming — only the typing indicator appears until the first token
  arrives.

### Security

- **Plugin path validation** is now enforced consistently on install, lookup,
  removal, and manifest load, blocking path-traversal via crafted plugin names.
- **Sandbox capability enforcement** — declared filesystem write paths are now
  granted explicitly. Browser plugins run under a write-confined relaxed sandbox
  (the browser engine needs broad syscall access, but writes are still limited
  to the plugin's own directories and the browser's scratch locations).
- **Scrape source validation** — unknown or disabled sources are rejected with a
  clear error instead of being silently skipped.
- **Install consent** — installing a plugin through the local API now requires
  explicit acknowledgement of its declared capabilities.
- Bumped `vite` to 6.4.3 and `esbuild` to ≥0.28.1 to clear high-severity
  advisories.

## [0.1.0] - 2026-06-12

### Added

- **One view of your work** — collect from every connected source in parallel
  (`aide run`) and triage a focused ACTION REQUIRED / INFORMATIONAL report
  (`aide report`), all stored locally under `~/.aide`.
- **Agent mode** — ask one-shot questions (`aide agent ask`) or run an autonomous
  assistant with a web chat UI (`aide agent start`), pointed at any LLM endpoint
  you trust.
- **Plugins as sandboxed sources** — install from a registry or a local path
  (`aide plugin install`); each plugin runs isolated per-OS with deny-by-default
  network access.
- **Bring your own sources** — the `aide dev` toolkit scaffolds, tests, and
  packages Python or Go plugins, and you can serve them from your own private
  registry.
- **Secure credentials** — secrets are kept in your OS keychain and managed with
  `aide credential`.
- **Works behind corporate TLS** — `--verify-ssl` / `--ca-bundle` with automatic
  OS trust-store support, propagated to plugins.
- **Prebuilt binaries** for macOS, Linux, and Windows, with built-in self-update.

[Unreleased]: https://github.com/matheus-meneses/aide/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/matheus-meneses/aide/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/matheus-meneses/aide/releases/tag/v0.1.0
