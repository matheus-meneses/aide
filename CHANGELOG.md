# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0]

0.2.0 brings a **native macOS desktop app** and a streamlined CLI — a new
`aide ui` launcher and a consolidated `aide plugin` command tree — alongside a
round of plugin/sandbox security hardening.

### Added

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

[Unreleased]: https://github.com/matheus-meneses/aide/compare/v0.2.0-rc.4...HEAD
[0.2.0]: https://github.com/matheus-meneses/aide/compare/v0.1.0...v0.2.0-rc.4
[0.1.0]: https://github.com/matheus-meneses/aide/releases/tag/v0.1.0
