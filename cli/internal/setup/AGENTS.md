# setup

## Purpose

First-run and ongoing provisioning flows that wire the lower concepts together: bootstrapping the
environment and provisioning sources/providers.

## Packages

- `bootstrap` — environment bootstrap (Python runtime, plugin prerequisites).
- `provision` — source/provider provisioning flows (config + credentials + plugin resolution).

## Dependency rules (depguard)

- **May import:** the Go standard library, third-party libs, `platform/*`, `persistence/*`,
  `security/*`, and `runtime/*`.
- **Must NOT import:** `notification`, `agent`, `ui`.
- `agent/api` is allowed to import `setup` (setup sits below agent's HTTP layer).
