# persistence

## Purpose

Durable storage. An inert leaf concept: it knows how to persist and query data, but never
orchestrates other subsystems.

## Packages

- `store` — SQLite persistence for items, metrics, team, and chat sessions (`store.Store`).

## Dependency rules (depguard)

- **May import:** the Go standard library and third-party libs (e.g. `modernc.org/sqlite`).
- **Must NOT import:** any other concept, including `platform`. `persistence` is a self-contained
  leaf so it can be reused without dragging in the rest of the tree.
