# platform

## Purpose

Inert, foundational primitives shared by every other concept. Packages here are leaves:
they do minimal OS work (path resolution, config file IO, logging sink) and contain **no
orchestration**.

## Packages

- `xdg` — platform-specific data/config/cache paths (`AideHome`, etc.).
- `clog` — scoped logging sink (stderr/file + live log subscribers).
- `config` — `AIDE_HOME`-rooted config loading/saving (`config.Config`).

`config` may import `xdg`. Otherwise these are independent.

## Dependency rules (depguard)

- **May import:** the Go standard library, third-party libs, and sibling `platform/*` packages.
- **Must NOT import:** any other concept (`persistence`, `security`, `runtime`, `setup`,
  `notification`, `agent`, `ui`). `platform` sits at the bottom of the dependency DAG.
