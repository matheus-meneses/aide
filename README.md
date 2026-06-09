# aide

aide is a local-first CLI that runs sandboxed plugins to aggregate work signals — Jira tickets, Outlook calendar events, GitLab merge requests, HR portal absences, identity approvals, and more — into a single terminal view. It also ships an autonomous agent with a web UI that monitors your signals and answers questions about your team.

## Install

**From a release binary** (macOS/Linux):

```bash
curl -fsSL https://raw.githubusercontent.com/matheus-meneses/aide/main/install.sh | bash
```

**From source:**

```bash
git clone https://github.com/matheus-meneses/aide.git
cd aide
make build
# binary at bin/aide
```

## Quickstart

```bash
aide init                      # first-time setup: downloads Python, initialises ~/.aide
aide plugin install jira       # install a plugin from the registry
aide config source add         # interactive wizard to configure the source
aide run                       # execute all enabled sources
aide report                    # view open items
aide agent start               # start the autonomous agent + web UI
```

## Architecture

```
aide (Go CLI binary)
├── cmd/aide/          Cobra command definitions (thin wrappers)
├── internal/
│   ├── plugin/        Plugin lifecycle: install, execute, sandbox
│   ├── runner/        Concurrent source execution, store integration
│   ├── store/         SQLite-backed item/metric/team store
│   ├── render/        Terminal output rendering
│   ├── agent/         Autonomous agent server + embedded React UI
│   ├── keychain/      Per-OS credential storage
│   └── ...
└── sdk/python/        Python plugin SDK (aide_sdk)

aide-plugins/          Plugin registry + builtin plugin sources
```

**Three runtimes:**

1. **Go CLI** — single static binary, cross-compiled for darwin/linux/windows (amd64 + arm64).
2. **Python plugin subprocess** — each plugin runs in an isolated `.venv`, invoked by aide with per-OS sandboxing (`sandbox-exec` on macOS, `bwrap` on Linux). Communication is a stdin/stdout JSON protocol.
3. **Embedded React UI** — built with Vite, embedded in the binary via `go:embed`, served by the agent when `aide agent start` runs.

## Plugin model

Plugins are self-contained directories with a `plugin.yaml` manifest, a Python scraper extending `BaseScraper`, and a `requirements.txt`. Install from the registry or a local path:

```bash
aide plugin install jira
aide plugin install --local path/to/my-plugin
```

See [aide-plugins](https://github.com/matheus-meneses/aide-plugins) for the builtin registry and [AGENTS.md](AGENTS.md) for authoring guidance.

## Development

```bash
make dev        # builds the binary and sets up ~/.aide-sandbox with AIDE_HOME + AIDE_SDK_PATH
make verify     # full polyglot gate: Go lint/test/vuln + Python lint/type + frontend typecheck/lint
make fmt        # auto-format Go (gofumpt), Python (ruff), and frontend (prettier)
```

Requirements: Go 1.26+, Python 3.11+, Node 18+.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full development guide.

## License

Apache License 2.0 — see [LICENSE](LICENSE).
