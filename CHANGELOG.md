# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-06-11

### Added

- Local-first work assistant: collects from every enabled source in parallel
  (`aide run`) and stores everything in a local SQLite database under `~/.aide`.
- Reporting (`aide report`) with an ACTION REQUIRED / INFORMATIONAL split.
- Autonomous agent with an embedded web UI (`aide agent start`) and one-shot
  questions (`aide agent ask`), talking to a configurable LLM endpoint.
- Plugin system running each plugin as an OS-sandboxed subprocess
  (`sandbox-exec` on macOS, `bwrap` on Linux) with deny-by-default filesystem
  and network access driven by `capabilities.network`.
- Plugin registry install (`aide plugin install [name[@version]]`), local
  install (`--local`), `list`, `remove`, `update`, and browser `auth`.
- Plugin development toolkit (`aide dev`): scaffold, validate, test, and package
  Python or Go plugins, with `--json` output for automation.
- Credential management backed by the OS keychain (macOS/Windows) or a protected
  local file (Linux): `aide credential set/show/list/delete`.
- TLS policy controls (`--verify-ssl`, `--ca-bundle`) propagated to plugins, with
  automatic OS trust-store support.
- Self-update check against GitHub releases (skipped for `dev` builds; override
  the source with `AIDE_RELEASE_URL`).
- Cross-platform release binaries for `darwin/amd64`, `darwin/arm64`,
  `linux/amd64`, `linux/arm64`, and `windows/amd64`, each with a SHA-256 checksum.

[Unreleased]: https://github.com/matheus-meneses/aide/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/matheus-meneses/aide/releases/tag/v0.1.0
