# runtime

## Purpose

The active subsystems that do real I/O: spawning processes, running plugins, scheduling scrapes,
provisioning Python, and self-updating. Unlike the leaf concepts, code here is allowed to touch the
network, filesystem, and child processes.

## Packages

- `exec` — process-group management for clean subprocess teardown (was `procctl`).
- `plugin` — plugin lifecycle: resolve, install, execute (consumes `security/sandbox` policy).
- `runner` — parallel scrape scheduling and result normalisation (semaphore concurrency).
- `pyenv` — Python virtualenv provisioning.
- `updater` — self-update from GitHub releases.

## Dependency rules (depguard)

- **May import:** the Go standard library, third-party libs, `platform/*`, `persistence/*`,
  `security/*`, and sibling `runtime/*` packages.
- **Must NOT import:** `setup`, `notification`, `agent`, `ui`.
