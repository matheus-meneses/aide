# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/matheus-meneses/aide/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/matheus-meneses/aide/releases/tag/v0.1.0
