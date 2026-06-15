# security

## Purpose

Credential storage and process isolation primitives.

## Packages

- `keychain` — per-OS credential storage (macOS Keychain, Linux secret tools, Windows), with
  build-tag-suffixed implementations (`keychain_darwin.go`, `keychain_linux.go`, …).
- `sandbox` — per-OS plugin sandbox policy (`sandbox_darwin.go`, etc.). Exposes a small `Policy`
  type so `runtime/plugin` can build a sandbox without `security` depending on plugin internals.

## Dependency rules (depguard)

- **May import:** the Go standard library, third-party libs, and `platform/*`.
- **Must NOT import:** `persistence`, `runtime`, `setup`, `notification`, `agent`, `ui`.
